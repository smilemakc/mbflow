package credentials

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// Service provides methods to work with credentials in workflows
type Service struct {
	repo       repository.CredentialsRepository
	encryption *crypto.EncryptionService
}

// NewService creates a new credentials service
func NewService(repo repository.CredentialsRepository, encryption *crypto.EncryptionService) *Service {
	return &Service{
		repo:       repo,
		encryption: encryption,
	}
}

// GetDecrypted retrieves a credential and decrypts its data
// This method should be used by executors to access credential values
func (s *Service) GetDecrypted(ctx context.Context, resourceID string) (*models.CredentialsResource, error) {
	cred, err := s.repo.GetCredentials(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	// Check if expired
	if cred.IsExpired() {
		return nil, fmt.Errorf("credential %s has expired", resourceID)
	}

	// Decrypt all data
	decrypted, err := s.encryption.DecryptMap(cred.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	cred.DecryptedData = decrypted

	// Increment usage counter (non-blocking)
	go func() {
		_ = s.repo.IncrementUsageCount(context.Background(), resourceID)
	}()

	return cred, nil
}

// GetAPIKey retrieves an API key credential and returns the key value
func (s *Service) GetAPIKey(ctx context.Context, resourceID string) (string, error) {
	cred, err := s.GetDecrypted(ctx, resourceID)
	if err != nil {
		return "", err
	}

	if cred.CredentialType != models.CredentialTypeAPIKey {
		return "", fmt.Errorf("credential %s is not an API key", resourceID)
	}

	return cred.GetAPIKey(), nil
}

// GetBasicAuth retrieves basic auth credentials
func (s *Service) GetBasicAuth(ctx context.Context, resourceID string) (username, password string, err error) {
	cred, err := s.GetDecrypted(ctx, resourceID)
	if err != nil {
		return "", "", err
	}

	if cred.CredentialType != models.CredentialTypeBasicAuth {
		return "", "", fmt.Errorf("credential %s is not basic auth", resourceID)
	}

	username, password = cred.GetBasicAuth()
	return username, password, nil
}

// GetOAuth2 retrieves OAuth2 credentials
func (s *Service) GetOAuth2(ctx context.Context, resourceID string) (*models.OAuth2Credential, error) {
	cred, err := s.GetDecrypted(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	if cred.CredentialType != models.CredentialTypeOAuth2 {
		return nil, fmt.Errorf("credential %s is not OAuth2", resourceID)
	}

	return cred.GetOAuth2(), nil
}

// GetServiceAccountJSON retrieves service account JSON
func (s *Service) GetServiceAccountJSON(ctx context.Context, resourceID string) (string, error) {
	cred, err := s.GetDecrypted(ctx, resourceID)
	if err != nil {
		return "", err
	}

	if cred.CredentialType != models.CredentialTypeServiceAccount {
		return "", fmt.Errorf("credential %s is not a service account", resourceID)
	}

	return cred.GetServiceAccountJSON(), nil
}

// GetCustomValue retrieves a custom field value from a credential
func (s *Service) GetCustomValue(ctx context.Context, resourceID, fieldName string) (string, error) {
	cred, err := s.GetDecrypted(ctx, resourceID)
	if err != nil {
		return "", err
	}

	value := cred.GetCustomValue(fieldName)
	if value == "" {
		return "", fmt.Errorf("field %q not found in credential %s", fieldName, resourceID)
	}

	return value, nil
}

// GetAllDecryptedValues retrieves all decrypted values as a map
func (s *Service) GetAllDecryptedValues(ctx context.Context, resourceID string) (map[string]string, error) {
	cred, err := s.GetDecrypted(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	return cred.DecryptedData, nil
}

// LogWorkflowUsage logs that a credential was used in a workflow execution
func (s *Service) LogWorkflowUsage(ctx context.Context, resourceID, workflowID, executionID string) error {
	metadata := map[string]any{
		"workflow_id":  workflowID,
		"execution_id": executionID,
	}
	return s.repo.LogCredentialAccess(ctx, resourceID, "used_in_workflow", "", "workflow", metadata)
}
