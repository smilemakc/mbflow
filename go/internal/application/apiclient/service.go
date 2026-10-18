// Package apiclient manages API clients imported from OpenAPI and Swagger specifications.
package apiclient

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/openapi"
)

const (
	DefaultListLimit = 50
	MaxListLimit     = 100

	maxNameLength        = 255
	maxDescriptionLength = 2000
	clientCacheSize      = 64
)

// CredentialLookup reads credentials without decrypting them. It is used to
// check that the credential selected for a client belongs to the client owner.
type CredentialLookup interface {
	GetCredentials(ctx context.Context, resourceID string) (*models.CredentialsResource, error)
}

// Service imports specifications and manages API client settings.
type Service struct {
	repo        repository.APIClientRepository
	credentials CredentialLookup
	cache       *clientCache
}

// NewService creates a Service. credentials may be nil when the server has
// no encryption key; clients then cannot reference credentials.
func NewService(repo repository.APIClientRepository, credentials CredentialLookup) *Service {
	return &Service{
		repo:        repo,
		credentials: credentials,
		cache:       newClientCache(clientCacheSize),
	}
}

// CreateInput describes a new client. Empty settings are taken from the specification.
type CreateInput struct {
	Name           string
	Description    string
	BaseURL        string
	Auth           *models.APIClientAuth
	DefaultHeaders map[string]string
	Spec           []byte
}

// UpdateInput changes client settings; nil fields stay unchanged.
type UpdateInput struct {
	Name           *string
	Description    *string
	BaseURL        *string
	Auth           *models.APIClientAuth
	DefaultHeaders map[string]string
}

// SpecChanges lists operation IDs added and removed by a specification update.
// Workflow nodes that reference a removed operation fail until reconfigured.
type SpecChanges struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
}

// Preview parses a specification without storing it.
func (s *Service) Preview(ctx context.Context, spec []byte) (*openapi.Spec, error) {
	return openapi.Parse(ctx, spec)
}

// Create imports a specification as a new client of the owner.
func (s *Service) Create(ctx context.Context, ownerID string, in CreateInput) (*models.APIClient, error) {
	spec, err := openapi.Parse(ctx, in.Spec)
	if err != nil {
		return nil, err
	}

	client := &models.APIClient{
		OwnerID:        ownerID,
		Name:           truncate(firstNonEmpty(in.Name, spec.Title), maxNameLength),
		Description:    truncate(firstNonEmpty(in.Description, spec.Description), maxDescriptionLength),
		BaseURL:        firstNonEmpty(in.BaseURL, spec.DefaultBaseURL()),
		Auth:           spec.SuggestedAuth,
		DefaultHeaders: in.DefaultHeaders,
	}
	if in.Auth != nil {
		client.Auth = *in.Auth
	}
	applySpec(client, spec, in.Spec)

	if err := s.validateSettings(ctx, client); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

// Get returns a client of the owner. Clients of other users are reported as not found.
func (s *Service) Get(ctx context.Context, ownerID, id string) (*models.APIClient, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, models.ErrInvalidID
	}
	client, err := s.repo.FindByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if client.OwnerID != ownerID {
		return nil, models.ErrAPIClientNotFound
	}
	return client, nil
}

// List returns the clients of the owner without operations and schemas.
func (s *Service) List(ctx context.Context, ownerID string, limit, offset int) ([]*models.APIClient, int64, error) {
	owner, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, 0, models.ErrInvalidID
	}
	if limit <= 0 {
		limit = DefaultListLimit
	}
	if limit > MaxListLimit {
		limit = MaxListLimit
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.FindByOwner(ctx, owner, limit, offset)
}

// Update changes client settings.
func (s *Service) Update(ctx context.Context, ownerID, id string, in UpdateInput) (*models.APIClient, error) {
	client, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		client.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		client.Description = truncate(*in.Description, maxDescriptionLength)
	}
	if in.BaseURL != nil {
		client.BaseURL = strings.TrimRight(strings.TrimSpace(*in.BaseURL), "/")
	}
	if in.Auth != nil {
		client.Auth = *in.Auth
	}
	if in.DefaultHeaders != nil {
		client.DefaultHeaders = in.DefaultHeaders
	}

	if err := s.validateSettings(ctx, client); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, client); err != nil {
		return nil, err
	}
	s.cache.remove(client.ID)
	return client, nil
}

// ReplaceSpec re-imports the specification of a client and keeps its settings.
// Operation IDs that stay in the new document keep working in existing workflows.
func (s *Service) ReplaceSpec(ctx context.Context, ownerID, id string, data []byte) (*models.APIClient, *SpecChanges, error) {
	client, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return nil, nil, err
	}
	spec, err := openapi.Parse(ctx, data)
	if err != nil {
		return nil, nil, err
	}

	previous := operationIDs(client.Operations)
	applySpec(client, spec, data)
	if client.BaseURL == "" {
		client.BaseURL = spec.DefaultBaseURL()
	}

	// The credential was checked when it was selected; a credential deleted
	// since then must not block a specification update.
	if err := client.Validate(); err != nil {
		return nil, nil, err
	}
	if err := s.repo.Update(ctx, client); err != nil {
		return nil, nil, err
	}
	s.cache.remove(client.ID)

	return client, diffOperations(previous, operationIDs(client.Operations)), nil
}

// GetSpec returns the original document of a client of the owner.
func (s *Service) GetSpec(ctx context.Context, ownerID, id string) (*models.APIClient, string, error) {
	client, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return nil, "", err
	}
	spec, err := s.repo.FindSpec(ctx, uuid.MustParse(client.ID))
	if err != nil {
		return nil, "", err
	}
	return client, spec, nil
}

// Delete removes a client of the owner.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	client, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, uuid.MustParse(client.ID)); err != nil {
		return err
	}
	s.cache.remove(client.ID)
	return nil
}

// GetAPIClient loads a client for workflow execution. It does not check the
// owner: the executor compares it with the workflow owner. Clients are cached
// until they change and are shared between executions, so callers must not
// modify them.
func (s *Service) GetAPIClient(ctx context.Context, id string) (*models.APIClient, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, models.ErrAPIClientNotFound
	}
	updatedAt, err := s.repo.FindUpdatedAt(ctx, uid)
	if err != nil {
		return nil, err
	}
	if client := s.cache.get(uid.String(), updatedAt); client != nil {
		return client, nil
	}
	client, err := s.repo.FindByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	s.cache.put(client)
	return client, nil
}

func (s *Service) validateSettings(ctx context.Context, client *models.APIClient) error {
	if err := client.Validate(); err != nil {
		return err
	}
	if client.Auth.CredentialID == "" {
		return nil
	}
	return s.checkCredential(ctx, client.OwnerID, client.Auth)
}

// compatibleCredentialTypes lists the credential types that hold the secret an auth type needs.
var compatibleCredentialTypes = map[models.APIClientAuthType][]models.CredentialType{
	models.APIClientAuthAPIKey:                  {models.CredentialTypeAPIKey, models.CredentialTypeOAuth2, models.CredentialTypeCustom},
	models.APIClientAuthBearer:                  {models.CredentialTypeAPIKey, models.CredentialTypeOAuth2, models.CredentialTypeCustom},
	models.APIClientAuthBasic:                   {models.CredentialTypeBasicAuth, models.CredentialTypeCustom},
	models.APIClientAuthOAuth2ClientCredentials: {models.CredentialTypeOAuth2, models.CredentialTypeCustom},
}

func (s *Service) checkCredential(ctx context.Context, ownerID string, auth models.APIClientAuth) error {
	const field = "auth.credential_id"
	if s.credentials == nil {
		return &models.ValidationError{Field: field, Message: "credentials are not available: encryption key is not configured"}
	}
	cred, err := s.credentials.GetCredentials(ctx, auth.CredentialID)
	if errors.Is(err, models.ErrResourceNotFound) || errors.Is(err, models.ErrInvalidID) || (err == nil && cred.OwnerID != ownerID) {
		return &models.ValidationError{Field: field, Message: "credential not found"}
	}
	if err != nil {
		return err
	}
	if cred.Status != models.ResourceStatusActive {
		return &models.ValidationError{Field: field, Message: "credential is not active"}
	}
	allowed, ok := compatibleCredentialTypes[auth.Type]
	if !ok {
		return &models.ValidationError{Field: field, Message: "auth type " + string(auth.Type) + " does not use credentials"}
	}
	for _, t := range allowed {
		if cred.CredentialType == t {
			return nil
		}
	}
	return &models.ValidationError{
		Field:   field,
		Message: "credential of type " + string(cred.CredentialType) + " cannot be used for " + string(auth.Type) + " auth",
	}
}

func applySpec(client *models.APIClient, spec *openapi.Spec, raw []byte) {
	client.Title = spec.Title
	client.APIVersion = spec.APIVersion
	client.SpecVersion = spec.SpecVersion
	client.Servers = spec.Servers
	client.SecuritySchemes = spec.SecuritySchemes
	client.Operations = spec.Operations
	client.Schemas = spec.Schemas
	client.Warnings = spec.Warnings
	client.OperationCount = len(spec.Operations)
	client.SpecRaw = string(raw)
	client.SpecSize = len(raw)
}

func operationIDs(ops []*models.APIOperation) map[string]bool {
	ids := make(map[string]bool, len(ops))
	for _, op := range ops {
		ids[op.ID] = true
	}
	return ids
}

func diffOperations(previous, current map[string]bool) *SpecChanges {
	changes := &SpecChanges{Added: []string{}, Removed: []string{}}
	for id := range current {
		if !previous[id] {
			changes.Added = append(changes.Added, id)
		}
	}
	for id := range previous {
		if !current[id] {
			changes.Removed = append(changes.Removed, id)
		}
	}
	sort.Strings(changes.Added)
	sort.Strings(changes.Removed)
	return changes
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxRunes]))
}

// clientCache keeps recently executed clients. An entry is valid while the
// stored updated_at matches, so edits made by other server instances are
// picked up on the next execution.
type clientCache struct {
	mu      sync.Mutex
	limit   int
	entries map[string]*models.APIClient
}

func newClientCache(limit int) *clientCache {
	return &clientCache{limit: limit, entries: make(map[string]*models.APIClient)}
}

func (c *clientCache) get(id string, updatedAt time.Time) *models.APIClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	if client, ok := c.entries[id]; ok && client.UpdatedAt.Equal(updatedAt) {
		return client
	}
	return nil
}

func (c *clientCache) put(client *models.APIClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[client.ID]; !exists && len(c.entries) >= c.limit {
		for id := range c.entries {
			delete(c.entries, id)
			break
		}
	}
	c.entries[client.ID] = client
}

func (c *clientCache) remove(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, id)
}
