package serviceapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// CredentialInfo represents a credential with its field keys (no secret values).
type CredentialInfo struct {
	ID             string
	Name           string
	Description    string
	Status         string
	CredentialType string
	Provider       string
	ExpiresAt      *time.Time
	LastUsedAt     *time.Time
	UsageCount     int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Fields         []string
}

func toCredentialInfo(cred *models.CredentialsResource) *CredentialInfo {
	fields := make([]string, 0, len(cred.EncryptedData))
	for k := range cred.EncryptedData {
		fields = append(fields, k)
	}

	return &CredentialInfo{
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

// ListCredentialsParams contains parameters for listing credentials.
type ListCredentialsParams struct {
	UserID   string
	Provider string
}

// ListCredentialsResult contains the result of listing credentials.
type ListCredentialsResult struct {
	Credentials []*CredentialInfo
}

func (o *Operations) ListCredentials(ctx context.Context, params ListCredentialsParams) (*ListCredentialsResult, error) {
	if params.UserID == "" {
		return nil, NewValidationError("USER_ID_REQUIRED", "user_id query parameter is required")
	}

	var credentials []*models.CredentialsResource
	var err error

	if params.Provider != "" {
		credentials, err = o.CredentialsRepo.GetCredentialsByProvider(ctx, params.UserID, params.Provider)
	} else {
		credentials, err = o.CredentialsRepo.GetCredentialsByOwner(ctx, params.UserID)
	}

	if err != nil {
		o.Logger.Error("Failed to list credentials", "error", err, "user_id", params.UserID)
		return nil, err
	}

	result := make([]*CredentialInfo, len(credentials))
	for i, cred := range credentials {
		result[i] = toCredentialInfo(cred)
	}

	return &ListCredentialsResult{Credentials: result}, nil
}

// CreateCredentialParams contains parameters for creating a credential.
type CreateCredentialParams struct {
	UserID         string
	Name           string
	Description    string
	CredentialType string
	Provider       string
	Data           map[string]string
}

func (o *Operations) CreateCredential(ctx context.Context, params CreateCredentialParams) (*CredentialInfo, error) {
	if params.UserID == "" {
		return nil, NewValidationError("UNAUTHORIZED", "unauthorized")
	}

	credType := models.CredentialType(params.CredentialType)
	if !models.IsValidCredentialType(credType) {
		return nil, NewValidationError("INVALID_CREDENTIAL_TYPE", "invalid credential_type")
	}

	encryptedData, err := o.EncryptionSvc.EncryptMap(params.Data)
	if err != nil {
		o.Logger.Error("Failed to encrypt credential data", "error", err, "user_id", params.UserID)
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	cred := models.NewCredentialsResource(params.UserID, params.Name, credType)
	cred.Description = params.Description
	cred.Provider = params.Provider
	cred.EncryptedData = encryptedData

	if err := o.CredentialsRepo.CreateCredentials(ctx, cred); err != nil {
		o.Logger.Error("Failed to create credential", "error", err, "user_id", params.UserID)
		return nil, err
	}

	o.Logger.Info("Credential created via service API",
		"credential_id", cred.ID,
		"user_id", params.UserID,
		"name", cred.Name,
		"credential_type", params.CredentialType,
	)

	return toCredentialInfo(cred), nil
}

// UpdateCredentialParams contains parameters for updating a credential.
type UpdateCredentialParams struct {
	CredentialID string
	Name         string
	Description  string
}

func (o *Operations) UpdateCredential(ctx context.Context, params UpdateCredentialParams) (*CredentialInfo, error) {
	cred, err := o.CredentialsRepo.GetCredentials(ctx, params.CredentialID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			return nil, models.ErrResourceNotFound
		}
		o.Logger.Error("Failed to get credential", "error", err, "credential_id", params.CredentialID)
		return nil, err
	}

	if params.Name != "" {
		cred.Name = params.Name
	}
	cred.Description = params.Description
	cred.UpdatedAt = time.Now()

	if err := o.CredentialsRepo.UpdateCredentials(ctx, cred); err != nil {
		o.Logger.Error("Failed to update credential", "error", err, "credential_id", params.CredentialID)
		return nil, err
	}

	o.Logger.Info("Credential updated via service API", "credential_id", params.CredentialID)
	return toCredentialInfo(cred), nil
}

// DeleteCredentialParams contains parameters for deleting a credential.
type DeleteCredentialParams struct {
	CredentialID string
}

func (o *Operations) DeleteCredential(ctx context.Context, params DeleteCredentialParams) error {
	_, err := o.CredentialsRepo.GetCredentials(ctx, params.CredentialID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			return models.ErrResourceNotFound
		}
		o.Logger.Error("Failed to get credential", "error", err, "credential_id", params.CredentialID)
		return err
	}

	credentialUUID, err := uuid.Parse(params.CredentialID)
	if err != nil {
		return NewValidationError("INVALID_ID", "invalid credential ID")
	}

	detachedCount, err := o.WorkflowRepo.UnassignResourceFromAllWorkflows(ctx, credentialUUID)
	if err != nil {
		o.Logger.Error("Failed to detach credential from workflows", "error", err, "credential_id", params.CredentialID)
		return err
	}

	if detachedCount > 0 {
		o.Logger.Info("Credential detached from workflows", "credential_id", params.CredentialID, "workflows_count", detachedCount)
	}

	if err := o.CredentialsRepo.DeleteCredentials(ctx, params.CredentialID); err != nil {
		o.Logger.Error("Failed to delete credential", "error", err, "credential_id", params.CredentialID)
		return err
	}

	o.Logger.Info("Credential deleted via service API",
		"credential_id", params.CredentialID,
		"detached_from_workflows", detachedCount,
	)

	return nil
}
