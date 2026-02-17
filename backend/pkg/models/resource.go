package models

import "time"

// ResourceType определяет тип ресурса
type ResourceType string

const (
	ResourceTypeFileStorage ResourceType = "file_storage"
	ResourceTypeCredentials ResourceType = "credentials"
	ResourceTypeRentalKey   ResourceType = "rental_key"
)

// ResourceStatus статус ресурса
type ResourceStatus string

const (
	ResourceStatusActive    ResourceStatus = "active"
	ResourceStatusSuspended ResourceStatus = "suspended"
	ResourceStatusDeleted   ResourceStatus = "deleted"
)

// Resource интерфейс для всех типов ресурсов
type Resource interface {
	GetID() string
	GetType() ResourceType
	GetOwnerID() string
	GetName() string
	GetDescription() string
	GetStatus() ResourceStatus
	GetMetadata() map[string]any
	Validate() error
}

// BaseResource базовая структура с общими полями для всех ресурсов
type BaseResource struct {
	ID          string         `json:"id"`
	Type        ResourceType   `json:"type"`
	OwnerID     string         `json:"owner_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Status      ResourceStatus `json:"status"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// GetID returns the resource ID
func (r *BaseResource) GetID() string {
	return r.ID
}

// GetType returns the resource type
func (r *BaseResource) GetType() ResourceType {
	return r.Type
}

// GetOwnerID returns the owner ID
func (r *BaseResource) GetOwnerID() string {
	return r.OwnerID
}

// GetName returns the resource name
func (r *BaseResource) GetName() string {
	return r.Name
}

// GetDescription returns the resource description
func (r *BaseResource) GetDescription() string {
	return r.Description
}

// GetStatus returns the resource status
func (r *BaseResource) GetStatus() ResourceStatus {
	return r.Status
}

// GetMetadata returns the resource metadata
func (r *BaseResource) GetMetadata() map[string]any {
	return r.Metadata
}

// Validate validates the base resource structure
func (r *BaseResource) Validate() error {
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "resource name is required"}
	}
	if r.OwnerID == "" {
		return &ValidationError{Field: "owner_id", Message: "owner ID is required"}
	}
	return nil
}

// IsActive checks if the resource is active
func (r *BaseResource) IsActive() bool {
	return r.Status == ResourceStatusActive
}
