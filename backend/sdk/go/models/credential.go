package models

import "time"

// Credential represents stored credentials for accessing external services.
type Credential struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}
