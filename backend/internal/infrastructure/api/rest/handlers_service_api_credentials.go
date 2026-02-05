package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

type ServiceAPICredentialHandlers struct {
	ops *serviceapi.Operations
}

func NewServiceAPICredentialHandlers(ops *serviceapi.Operations) *ServiceAPICredentialHandlers {
	return &ServiceAPICredentialHandlers{ops: ops}
}

func (h *ServiceAPICredentialHandlers) ListCredentials(c *gin.Context) {
	result, err := h.ops.ListCredentials(c.Request.Context(), serviceapi.ListCredentialsParams{
		UserID:   c.Query("user_id"),
		Provider: c.Query("provider"),
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"credentials": result.Credentials})
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

	cred, err := h.ops.CreateCredential(c.Request.Context(), serviceapi.CreateCredentialParams{
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		CredentialType: req.CredentialType,
		Provider:       req.Provider,
		Data:           req.Data,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	respondJSON(c, http.StatusCreated, cred)
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

	cred, err := h.ops.UpdateCredential(c.Request.Context(), serviceapi.UpdateCredentialParams{
		CredentialID: credentialID,
		Name:         req.Name,
		Description:  req.Description,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	c.JSON(http.StatusOK, cred)
}

func (h *ServiceAPICredentialHandlers) DeleteCredential(c *gin.Context) {
	credentialID, ok := getParam(c, "id")
	if !ok {
		return
	}

	if err := h.ops.DeleteCredential(c.Request.Context(), serviceapi.DeleteCredentialParams{
		CredentialID: credentialID,
	}); err != nil {
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "credential deleted successfully"})
}
