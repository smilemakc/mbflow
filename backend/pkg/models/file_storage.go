package models

import (
	"fmt"
	"time"
)

// AccessScope defines the visibility scope of a file in the storage
type AccessScope string

const (
	// ScopeWorkflow - file is accessible within the entire workflow
	ScopeWorkflow AccessScope = "workflow"
	// ScopeEdge - file is accessible only between connected nodes
	ScopeEdge AccessScope = "edge"
	// ScopeResult - file is stored as node execution result
	ScopeResult AccessScope = "result"
	// ScopeResource - file belongs to a FileStorage resource
	ScopeResource AccessScope = "resource"
)

// ValidAccessScopes contains all valid access scope values
var ValidAccessScopes = map[AccessScope]bool{
	ScopeWorkflow: true,
	ScopeEdge:     true,
	ScopeResult:   true,
	ScopeResource: true,
}

// IsValid checks if the access scope is valid
func (s AccessScope) IsValid() bool {
	return ValidAccessScopes[s]
}

// FileEntry represents a file stored in the file storage system
type FileEntry struct {
	ID           string                 `json:"id"`
	StorageID    string                 `json:"storage_id"`     // Storage identifier
	Name         string                 `json:"name"`           // Original file name
	Path         string                 `json:"path"`           // Path within storage
	MimeType     string                 `json:"mime_type"`      // MIME type
	Size         int64                  `json:"size"`           // Size in bytes
	Checksum     string                 `json:"checksum"`       // SHA256 checksum
	AccessScope  AccessScope            `json:"access_scope"`   // Access scope
	Tags         []string               `json:"tags,omitempty"` // Tags for filtering
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	TTL          *time.Duration         `json:"ttl,omitempty"`         // Time to live
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`  // Expiration timestamp
	WorkflowID   *string                `json:"workflow_id,omitempty"` // Optional workflow reference
	ExecutionID  *string                `json:"execution_id,omitempty"`
	SourceNodeID *string                `json:"source_node_id,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// Validate validates the file entry
func (f *FileEntry) Validate() error {
	if f.ID == "" {
		return &ValidationError{Field: "id", Message: "file ID is required"}
	}
	if f.StorageID == "" {
		return &ValidationError{Field: "storage_id", Message: "storage ID is required"}
	}
	if f.Name == "" {
		return &ValidationError{Field: "name", Message: "file name is required"}
	}
	if f.MimeType == "" {
		return &ValidationError{Field: "mime_type", Message: "MIME type is required"}
	}
	if !f.AccessScope.IsValid() {
		return &ValidationError{Field: "access_scope", Message: fmt.Sprintf("invalid access scope: %s", f.AccessScope)}
	}
	if f.Size < 0 {
		return &ValidationError{Field: "size", Message: "file size cannot be negative"}
	}
	return nil
}

// IsExpired checks if the file has expired
func (f *FileEntry) IsExpired() bool {
	if f.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*f.ExpiresAt)
}

// SetTTL sets the TTL and calculates ExpiresAt
func (f *FileEntry) SetTTL(ttl time.Duration) {
	f.TTL = &ttl
	expiresAt := time.Now().Add(ttl)
	f.ExpiresAt = &expiresAt
}

// StorageConfig holds configuration for a storage instance
type StorageConfig struct {
	Type        StorageType            `json:"type"`              // Storage type (local, s3, etc.)
	BasePath    string                 `json:"base_path"`         // Base path for local storage
	MaxSize     int64                  `json:"max_size"`          // Maximum storage size in bytes (0 = unlimited)
	MaxFileSize int64                  `json:"max_file_size"`     // Maximum file size in bytes
	DefaultTTL  *time.Duration         `json:"default_ttl"`       // Default TTL for files
	Options     map[string]interface{} `json:"options,omitempty"` // Provider-specific options
}

// StorageType represents the type of storage backend
type StorageType string

const (
	StorageTypeLocal StorageType = "local"
	StorageTypeS3    StorageType = "s3"
	// Future: StorageTypeGCS, StorageTypeAzure, etc.
)

// StorageUsage contains storage usage statistics
type StorageUsage struct {
	StorageID    string  `json:"storage_id"`
	TotalSize    int64   `json:"total_size"`    // Total used size in bytes
	FileCount    int64   `json:"file_count"`    // Number of files
	MaxSize      int64   `json:"max_size"`      // Maximum allowed size (0 = unlimited)
	UsagePercent float64 `json:"usage_percent"` // Usage percentage
}

// AllowedMimeTypes defines the whitelist of allowed MIME types
var AllowedMimeTypes = map[string]bool{
	// Images
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
	"image/bmp":     true,
	"image/tiff":    true,

	// Documents
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,

	// Audio
	"audio/mpeg": true,
	"audio/wav":  true,
	"audio/ogg":  true,
	"audio/webm": true,
	"audio/flac": true,

	// Video
	"video/mp4":       true,
	"video/webm":      true,
	"video/ogg":       true,
	"video/mpeg":      true,
	"video/quicktime": true,

	// Text
	"text/plain":       true,
	"text/csv":         true,
	"text/html":        true,
	"text/markdown":    true,
	"application/json": true,
	"application/xml":  true,

	// Archives
	"application/zip":              true,
	"application/gzip":             true,
	"application/x-tar":            true,
	"application/x-rar-compressed": true,
	"application/x-7z-compressed":  true,
}

// IsMimeTypeAllowed checks if a MIME type is in the allowed list
func IsMimeTypeAllowed(mimeType string) bool {
	return AllowedMimeTypes[mimeType]
}
