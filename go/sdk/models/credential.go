// Manually maintained. Source: pkg/models/credentials_resource.go (flattened from CredentialsResource).
package models

import "time"

// Credential represents stored credentials for accessing external services.
// This is a flattened view of the backend's CredentialsResource, matching the API response shape.
type Credential struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	CredentialType string            `json:"credential_type"`
	Description    string            `json:"description,omitempty"`
	Status         string            `json:"status,omitempty"`
	Provider       string            `json:"provider,omitempty"`
	ExpiresAt      *time.Time        `json:"expires_at,omitempty"`
	LastUsedAt     *time.Time        `json:"last_used_at,omitempty"`
	UsageCount     int64             `json:"usage_count,omitempty"`
	Fields         []string          `json:"fields,omitempty"`
	Data           map[string]string `json:"data,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Metadata       map[string]any    `json:"metadata,omitempty"`
}
