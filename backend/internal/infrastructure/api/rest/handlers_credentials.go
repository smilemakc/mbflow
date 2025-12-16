package rest

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/crypto"
	"github.com/smilemakc/mbflow/pkg/models"
)

// CredentialsHandlers handles credentials-related HTTP requests
type CredentialsHandlers struct {
	credRepo     repository.CredentialsRepository
	workflowRepo repository.WorkflowRepository
	encryption   *crypto.EncryptionService
	logger       *logger.Logger
}

// NewCredentialsHandlers creates a new CredentialsHandlers instance
func NewCredentialsHandlers(credRepo repository.CredentialsRepository, workflowRepo repository.WorkflowRepository, encryption *crypto.EncryptionService, log *logger.Logger) *CredentialsHandlers {
	return &CredentialsHandlers{
		credRepo:     credRepo,
		workflowRepo: workflowRepo,
		encryption:   encryption,
		logger:       log,
	}
}

// ============================================================================
// Request/Response types
// ============================================================================

// CreateAPIKeyRequest represents request to create API key credential
type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
	Provider    string `json:"provider" binding:"max=100"`
	APIKey      string `json:"api_key" binding:"required"`
}

// CreateBasicAuthRequest represents request to create basic auth credential
type CreateBasicAuthRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
	Provider    string `json:"provider" binding:"max=100"`
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

// CreateOAuth2Request represents request to create OAuth2 credential
type CreateOAuth2Request struct {
	Name         string `json:"name" binding:"required,min=1,max=255"`
	Description  string `json:"description" binding:"max=1000"`
	Provider     string `json:"provider" binding:"max=100"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenURL     string `json:"token_url"`
	Scopes       string `json:"scopes"`
}

// CreateServiceAccountRequest represents request to create service account credential
type CreateServiceAccountRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
	Provider    string `json:"provider" binding:"max=100"`
	JSONKey     string `json:"json_key" binding:"required"` // Full JSON content
}

// CreateCustomCredentialRequest represents request to create custom credential
type CreateCustomCredentialRequest struct {
	Name        string            `json:"name" binding:"required,min=1,max=255"`
	Description string            `json:"description" binding:"max=1000"`
	Provider    string            `json:"provider" binding:"max=100"`
	Data        map[string]string `json:"data" binding:"required,min=1"`
}

// CredentialResponse represents a credential in API response (without secrets)
type CredentialResponse struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	CredentialType string     `json:"credential_type"`
	Provider       string     `json:"provider,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	UsageCount     int64      `json:"usage_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	// Fields list (keys only, no values)
	Fields []string `json:"fields,omitempty"`
}

// CredentialWithSecretsResponse includes decrypted values (used only for specific endpoints)
type CredentialWithSecretsResponse struct {
	CredentialResponse
	Data map[string]string `json:"data"`
}

// UpdateCredentialRequest represents request to update credential metadata
type UpdateCredentialRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
}

// ============================================================================
// Handlers
// ============================================================================

// CreateAPIKey creates a new API key credential
// POST /api/v1/credentials/api-key
func (h *CredentialsHandlers) CreateAPIKey(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateAPIKeyRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Encrypt the API key
	encryptedKey, err := h.encryption.EncryptString(req.APIKey)
	if err != nil {
		h.logger.Error("Failed to encrypt API key", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "encryption failed")
		return
	}

	cred := models.NewCredentialsResource(userID, req.Name, models.CredentialTypeAPIKey)
	cred.Description = req.Description
	cred.Provider = req.Provider
	cred.EncryptedData = map[string]string{"api_key": encryptedKey}

	if err := h.credRepo.CreateCredentials(c.Request.Context(), cred); err != nil {
		h.logger.Error("Failed to create API key credential", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to create credential")
		return
	}

	h.logger.Info("API key credential created",
		"credential_id", cred.ID,
		"user_id", userID,
		"name", cred.Name,
		"provider", cred.Provider,
	)

	respondJSON(c, http.StatusCreated, h.toResponse(cred))
}

// CreateBasicAuth creates a new basic auth credential
// POST /api/v1/credentials/basic-auth
func (h *CredentialsHandlers) CreateBasicAuth(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateBasicAuthRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Encrypt username and password
	encryptedData, err := h.encryption.EncryptMap(map[string]string{
		"username": req.Username,
		"password": req.Password,
	})
	if err != nil {
		h.logger.Error("Failed to encrypt basic auth", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "encryption failed")
		return
	}

	cred := models.NewCredentialsResource(userID, req.Name, models.CredentialTypeBasicAuth)
	cred.Description = req.Description
	cred.Provider = req.Provider
	cred.EncryptedData = encryptedData

	if err := h.credRepo.CreateCredentials(c.Request.Context(), cred); err != nil {
		h.logger.Error("Failed to create basic auth credential", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to create credential")
		return
	}

	h.logger.Info("Basic auth credential created",
		"credential_id", cred.ID,
		"user_id", userID,
		"name", cred.Name,
	)

	respondJSON(c, http.StatusCreated, h.toResponse(cred))
}

// CreateOAuth2 creates a new OAuth2 credential
// POST /api/v1/credentials/oauth2
func (h *CredentialsHandlers) CreateOAuth2(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateOAuth2Request
	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Build data map with non-empty fields
	data := map[string]string{
		"client_id":     req.ClientID,
		"client_secret": req.ClientSecret,
	}
	if req.AccessToken != "" {
		data["access_token"] = req.AccessToken
	}
	if req.RefreshToken != "" {
		data["refresh_token"] = req.RefreshToken
	}
	if req.TokenURL != "" {
		data["token_url"] = req.TokenURL
	}
	if req.Scopes != "" {
		data["scopes"] = req.Scopes
	}

	// Encrypt all fields
	encryptedData, err := h.encryption.EncryptMap(data)
	if err != nil {
		h.logger.Error("Failed to encrypt OAuth2 data", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "encryption failed")
		return
	}

	cred := models.NewCredentialsResource(userID, req.Name, models.CredentialTypeOAuth2)
	cred.Description = req.Description
	cred.Provider = req.Provider
	cred.EncryptedData = encryptedData

	if err := h.credRepo.CreateCredentials(c.Request.Context(), cred); err != nil {
		h.logger.Error("Failed to create OAuth2 credential", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to create credential")
		return
	}

	h.logger.Info("OAuth2 credential created",
		"credential_id", cred.ID,
		"user_id", userID,
		"name", cred.Name,
		"provider", cred.Provider,
	)

	respondJSON(c, http.StatusCreated, h.toResponse(cred))
}

// CreateServiceAccount creates a new service account credential
// POST /api/v1/credentials/service-account
func (h *CredentialsHandlers) CreateServiceAccount(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateServiceAccountRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Encrypt the JSON key
	encryptedKey, err := h.encryption.EncryptString(req.JSONKey)
	if err != nil {
		h.logger.Error("Failed to encrypt service account key", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "encryption failed")
		return
	}

	cred := models.NewCredentialsResource(userID, req.Name, models.CredentialTypeServiceAccount)
	cred.Description = req.Description
	cred.Provider = req.Provider
	cred.EncryptedData = map[string]string{"json_key": encryptedKey}

	if err := h.credRepo.CreateCredentials(c.Request.Context(), cred); err != nil {
		h.logger.Error("Failed to create service account credential", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to create credential")
		return
	}

	h.logger.Info("Service account credential created",
		"credential_id", cred.ID,
		"user_id", userID,
		"name", cred.Name,
		"provider", cred.Provider,
	)

	respondJSON(c, http.StatusCreated, h.toResponse(cred))
}

// CreateCustom creates a new custom credential
// POST /api/v1/credentials/custom
func (h *CredentialsHandlers) CreateCustom(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateCustomCredentialRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Encrypt all custom fields
	encryptedData, err := h.encryption.EncryptMap(req.Data)
	if err != nil {
		h.logger.Error("Failed to encrypt custom data", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "encryption failed")
		return
	}

	cred := models.NewCredentialsResource(userID, req.Name, models.CredentialTypeCustom)
	cred.Description = req.Description
	cred.Provider = req.Provider
	cred.EncryptedData = encryptedData

	if err := h.credRepo.CreateCredentials(c.Request.Context(), cred); err != nil {
		h.logger.Error("Failed to create custom credential", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to create credential")
		return
	}

	h.logger.Info("Custom credential created",
		"credential_id", cred.ID,
		"user_id", userID,
		"name", cred.Name,
	)

	respondJSON(c, http.StatusCreated, h.toResponse(cred))
}

// ListCredentials returns all credentials for the current user
// GET /api/v1/credentials
func (h *CredentialsHandlers) ListCredentials(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	provider := c.Query("provider")

	var credentials []*models.CredentialsResource
	var err error

	if provider != "" {
		credentials, err = h.credRepo.GetCredentialsByProvider(c.Request.Context(), userID, provider)
	} else {
		credentials, err = h.credRepo.GetCredentialsByOwner(c.Request.Context(), userID)
	}

	if err != nil {
		h.logger.Error("Failed to list credentials", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to list credentials")
		return
	}

	response := make([]CredentialResponse, len(credentials))
	for i, cred := range credentials {
		response[i] = h.toResponse(cred)
	}

	c.JSON(http.StatusOK, gin.H{"credentials": response})
}

// GetCredential returns a specific credential by ID (without secrets)
// GET /api/v1/credentials/:id
func (h *CredentialsHandlers) GetCredential(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	credentialID, ok := getParam(c, "id")
	if !ok {
		return
	}

	cred, err := h.credRepo.GetCredentials(c.Request.Context(), credentialID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "credential not found")
			return
		}
		h.logger.Error("Failed to get credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to get credential")
		return
	}

	if cred.OwnerID != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	c.JSON(http.StatusOK, h.toResponse(cred))
}

// GetCredentialSecrets returns a credential with decrypted secrets
// GET /api/v1/credentials/:id/secrets
// This is a sensitive endpoint - use with caution
func (h *CredentialsHandlers) GetCredentialSecrets(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	credentialID, ok := getParam(c, "id")
	if !ok {
		return
	}

	cred, err := h.credRepo.GetCredentials(c.Request.Context(), credentialID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "credential not found")
			return
		}
		h.logger.Error("Failed to get credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to get credential")
		return
	}

	if cred.OwnerID != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	// Check if credential is expired
	if cred.IsExpired() {
		respondError(c, http.StatusGone, "credential has expired")
		return
	}

	// Decrypt all data
	decryptedData, err := h.encryption.DecryptMap(cred.EncryptedData)
	if err != nil {
		h.logger.Error("Failed to decrypt credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "decryption failed")
		return
	}

	// Increment usage and log access
	if err := h.credRepo.IncrementUsageCount(c.Request.Context(), credentialID); err != nil {
		h.logger.Warn("Failed to increment usage count", "error", err, "credential_id", credentialID)
	}

	if err := h.credRepo.LogCredentialAccess(c.Request.Context(), credentialID, "read", userID, "user", nil); err != nil {
		h.logger.Warn("Failed to log credential access", "error", err, "credential_id", credentialID)
	}

	h.logger.Info("Credential secrets accessed",
		"credential_id", credentialID,
		"user_id", userID,
	)

	response := CredentialWithSecretsResponse{
		CredentialResponse: h.toResponse(cred),
		Data:               decryptedData,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateCredential updates credential metadata (not secrets)
// PUT /api/v1/credentials/:id
func (h *CredentialsHandlers) UpdateCredential(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	credentialID, ok := getParam(c, "id")
	if !ok {
		return
	}

	var req UpdateCredentialRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	cred, err := h.credRepo.GetCredentials(c.Request.Context(), credentialID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "credential not found")
			return
		}
		h.logger.Error("Failed to get credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to get credential")
		return
	}

	if cred.OwnerID != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	if req.Name != "" {
		cred.Name = req.Name
	}
	cred.Description = req.Description
	cred.UpdatedAt = time.Now()

	if err := h.credRepo.UpdateCredentials(c.Request.Context(), cred); err != nil {
		h.logger.Error("Failed to update credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to update credential")
		return
	}

	h.logger.Info("Credential updated",
		"credential_id", credentialID,
		"user_id", userID,
	)

	c.JSON(http.StatusOK, h.toResponse(cred))
}

// DeleteCredential soft-deletes a credential
// DELETE /api/v1/credentials/:id
func (h *CredentialsHandlers) DeleteCredential(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	credentialID, ok := getParam(c, "id")
	if !ok {
		return
	}

	cred, err := h.credRepo.GetCredentials(c.Request.Context(), credentialID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "credential not found")
			return
		}
		h.logger.Error("Failed to get credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to get credential")
		return
	}

	if cred.OwnerID != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	// First, detach credential from all workflows
	credentialUUID, err := uuid.Parse(credentialID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid credential ID")
		return
	}

	detachedCount, err := h.workflowRepo.UnassignResourceFromAllWorkflows(c.Request.Context(), credentialUUID)
	if err != nil {
		h.logger.Error("Failed to detach credential from workflows", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to detach credential from workflows")
		return
	}

	if detachedCount > 0 {
		h.logger.Info("Credential detached from workflows", "credential_id", credentialID, "workflows_count", detachedCount)
	}

	// Then delete the credential
	if err := h.credRepo.DeleteCredentials(c.Request.Context(), credentialID); err != nil {
		h.logger.Error("Failed to delete credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to delete credential")
		return
	}

	h.logger.Info("Credential deleted",
		"credential_id", credentialID,
		"user_id", userID,
		"detached_from_workflows", detachedCount,
	)

	c.JSON(http.StatusOK, gin.H{"message": "credential deleted successfully"})
}

// ============================================================================
// Helper methods
// ============================================================================

func (h *CredentialsHandlers) toResponse(cred *models.CredentialsResource) CredentialResponse {
	// Extract field names (keys only) from encrypted data
	fields := make([]string, 0, len(cred.EncryptedData))
	for k := range cred.EncryptedData {
		fields = append(fields, k)
	}

	return CredentialResponse{
		ID:             cred.ID,
		Name:           cred.Name,
		Description:    cred.Description,
		Status:         string(cred.Status),
		CredentialType: string(cred.CredentialType),
		Provider:       cred.Provider,
		ExpiresAt:      cred.ExpiresAt,
		LastUsedAt:     cred.LastUsedAt,
		UsageCount:     cred.UsageCount,
		CreatedAt:      cred.CreatedAt,
		UpdatedAt:      cred.UpdatedAt,
		Fields:         fields,
	}
}
