package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// FileModel represents a file entry in the database
type FileModel struct {
	bun.BaseModel `bun:"table:files,alias:f"`

	ID           uuid.UUID   `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	StorageID    string      `bun:"storage_id,notnull" json:"storage_id" validate:"required"`
	Name         string      `bun:"name,notnull" json:"name" validate:"required"`
	Path         string      `bun:"path,notnull" json:"path" validate:"required"`
	MimeType     string      `bun:"mime_type,notnull" json:"mime_type" validate:"required"`
	Size         int64       `bun:"size,notnull,default:0" json:"size"`
	Checksum     string      `bun:"checksum,notnull" json:"checksum" validate:"required"`
	AccessScope  string      `bun:"access_scope,notnull,default:'workflow'" json:"access_scope"`
	Tags         StringArray `bun:"tags,type:text[],default:'{}'" json:"tags,omitempty"`
	Metadata     JSONBMap    `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	TTLSeconds   *int        `bun:"ttl_seconds" json:"ttl_seconds,omitempty"`
	ExpiresAt    *time.Time  `bun:"expires_at" json:"expires_at,omitempty"`
	ResourceID   *uuid.UUID  `bun:"resource_id,type:uuid" json:"resource_id,omitempty"`
	WorkflowID   *uuid.UUID  `bun:"workflow_id,type:uuid" json:"workflow_id,omitempty"`
	ExecutionID  *uuid.UUID  `bun:"execution_id,type:uuid" json:"execution_id,omitempty"`
	SourceNodeID *string     `bun:"source_node_id" json:"source_node_id,omitempty"`
	CreatedAt    time.Time   `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt    time.Time   `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relationships
	Resource  *ResourceModel  `bun:"rel:belongs-to,join:resource_id=id" json:"resource,omitempty"`
	Workflow  *WorkflowModel  `bun:"rel:belongs-to,join:workflow_id=id" json:"workflow,omitempty"`
	Execution *ExecutionModel `bun:"rel:belongs-to,join:execution_id=id" json:"execution,omitempty"`
}

// TableName returns the table name
func (FileModel) TableName() string {
	return "files"
}

// BeforeInsert hook to set timestamps
func (f *FileModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	if f.Metadata == nil {
		f.Metadata = make(JSONBMap)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (f *FileModel) BeforeUpdate(ctx interface{}) error {
	f.UpdatedAt = time.Now()
	return nil
}

// IsExpired checks if the file has expired
func (f *FileModel) IsExpired() bool {
	if f.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*f.ExpiresAt)
}

// StorageConfigModel represents a storage configuration in the database
type StorageConfigModel struct {
	bun.BaseModel `bun:"table:storage_configs,alias:sc"`

	ID                uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	StorageID         string    `bun:"storage_id,notnull,unique" json:"storage_id" validate:"required"`
	StorageType       string    `bun:"storage_type,notnull,default:'local'" json:"storage_type"`
	Config            JSONBMap  `bun:"config,type:jsonb,default:'{}'" json:"config"`
	MaxSize           int64     `bun:"max_size,default:0" json:"max_size"`
	MaxFileSize       int64     `bun:"max_file_size,default:0" json:"max_file_size"`
	DefaultTTLSeconds *int      `bun:"default_ttl_seconds" json:"default_ttl_seconds,omitempty"`
	CreatedAt         time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt         time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// TableName returns the table name
func (StorageConfigModel) TableName() string {
	return "storage_configs"
}

// BeforeInsert hook to set timestamps
func (s *StorageConfigModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Config == nil {
		s.Config = make(JSONBMap)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (s *StorageConfigModel) BeforeUpdate(ctx interface{}) error {
	s.UpdatedAt = time.Now()
	return nil
}
