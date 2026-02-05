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

type ServiceAPICredentialHandlers struct {
	credentialsRepo repository.CredentialsRepository
	workflowRepo    repository.WorkflowRepository
	encryptionSvc   *crypto.EncryptionService
	logger          *logger.Logger
}

func NewServiceAPICredentialHandlers(
	credentialsRepo repository.CredentialsRepository,
	workflowRepo repository.WorkflowRepository,
	encryptionSvc *crypto.EncryptionService,
	log *logger.Logger,
) *ServiceAPICredentialHandlers {
	return &ServiceAPICredentialHandlers{
		credentialsRepo: credentialsRepo,
		workflowRepo:    workflowRepo,
		encryptionSvc:   encryptionSvc,
		logger:          log,
	}
}

func (h *ServiceAPICredentialHandlers) ListCredentials(c *gin.Context) {
	userIDParam := c.Query("user_id")
	provider := c.Query("provider")

	if userIDParam == "" {
		respondError(c, http.StatusBadRequest, "user_id query parameter is required")
		return
	}

	var credentials []*models.CredentialsResource
	var err error

	if provider != "" {
		credentials, err = h.credentialsRepo.GetCredentialsByProvider(c.Request.Context(), userIDParam, provider)
	} else {
		credentials, err = h.credentialsRepo.GetCredentialsByOwner(c.Request.Context(), userIDParam)
	}

	if err != nil {
		h.logger.Error("Failed to list credentials", "error", err, "user_id", userIDParam)
		respondError(c, http.StatusInternalServerError, "failed to list credentials")
		return
	}

	response := make([]CredentialResponse, len(credentials))
	for i, cred := range credentials {
		response[i] = toServiceAPICredentialResponse(cred)
	}

	c.JSON(http.StatusOK, gin.H{"credentials": response})
}

func (h *ServiceAPICredentialHandlers) CreateCredential(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name           string            `json:"name" binding:"required,min=1,max=255"`
		Description    string            `json:"description" binding:"max=1000"`
		CredentialType string            `json:"credential_type" binding:"required"`
		Provider       string            `json:"provider" binding:"max=100"`
		Data           map[string]string `json:"data" binding:"required,min=1"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	credType := models.CredentialType(req.CredentialType)
	if !models.IsValidCredentialType(credType) {
		respondError(c, http.StatusBadRequest, "invalid credential_type")
		return
	}

	encryptedData, err := h.encryptionSvc.EncryptMap(req.Data)
	if err != nil {
		h.logger.Error("Failed to encrypt credential data", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "encryption failed")
		return
	}

	cred := models.NewCredentialsResource(userID, req.Name, credType)
	cred.Description = req.Description
	cred.Provider = req.Provider
	cred.EncryptedData = encryptedData

	if err := h.credentialsRepo.CreateCredentials(c.Request.Context(), cred); err != nil {
		h.logger.Error("Failed to create credential", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to create credential")
		return
	}

	h.logger.Info("Credential created via service API",
		"credential_id", cred.ID,
		"user_id", userID,
		"name", cred.Name,
		"credential_type", req.CredentialType,
	)

	respondJSON(c, http.StatusCreated, toServiceAPICredentialResponse(cred))
}

func (h *ServiceAPICredentialHandlers) UpdateCredential(c *gin.Context) {
	credentialID, ok := getParam(c, "id")
	if !ok {
		return
	}

	var req UpdateCredentialRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	cred, err := h.credentialsRepo.GetCredentials(c.Request.Context(), credentialID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "credential not found")
			return
		}
		h.logger.Error("Failed to get credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to get credential")
		return
	}

	if req.Name != "" {
		cred.Name = req.Name
	}
	cred.Description = req.Description
	cred.UpdatedAt = time.Now()

	if err := h.credentialsRepo.UpdateCredentials(c.Request.Context(), cred); err != nil {
		h.logger.Error("Failed to update credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to update credential")
		return
	}

	h.logger.Info("Credential updated via service API", "credential_id", credentialID)

	c.JSON(http.StatusOK, toServiceAPICredentialResponse(cred))
}

func (h *ServiceAPICredentialHandlers) DeleteCredential(c *gin.Context) {
	credentialID, ok := getParam(c, "id")
	if !ok {
		return
	}

	_, err := h.credentialsRepo.GetCredentials(c.Request.Context(), credentialID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "credential not found")
			return
		}
		h.logger.Error("Failed to get credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to get credential")
		return
	}

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

	if err := h.credentialsRepo.DeleteCredentials(c.Request.Context(), credentialID); err != nil {
		h.logger.Error("Failed to delete credential", "error", err, "credential_id", credentialID)
		respondError(c, http.StatusInternalServerError, "failed to delete credential")
		return
	}

	h.logger.Info("Credential deleted via service API",
		"credential_id", credentialID,
		"detached_from_workflows", detachedCount,
	)

	c.JSON(http.StatusOK, gin.H{"message": "credential deleted successfully"})
}

func toServiceAPICredentialResponse(cred *models.CredentialsResource) CredentialResponse {
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
