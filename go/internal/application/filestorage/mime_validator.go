package filestorage

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// MimeValidator validates MIME types against a whitelist
type MimeValidator struct {
	allowedTypes map[string]bool
}

// NewMimeValidator creates a new MIME validator with default allowed types
func NewMimeValidator() *MimeValidator {
	return &MimeValidator{
		allowedTypes: models.AllowedMimeTypes,
	}
}

// NewMimeValidatorWithTypes creates a validator with custom allowed types
func NewMimeValidatorWithTypes(types []string) *MimeValidator {
	allowed := make(map[string]bool)
	for _, t := range types {
		allowed[t] = true
	}
	return &MimeValidator{
		allowedTypes: allowed,
	}
}

// IsAllowed checks if a MIME type is allowed
func (v *MimeValidator) IsAllowed(mimeType string) bool {
	// Normalize MIME type (remove parameters like charset)
	normalized := v.normalizeMimeType(mimeType)
	return v.allowedTypes[normalized]
}

// Validate validates a MIME type and returns an error if not allowed
func (v *MimeValidator) Validate(mimeType string) error {
	if !v.IsAllowed(mimeType) {
		return fmt.Errorf("MIME type not allowed: %s", mimeType)
	}
	return nil
}

// normalizeMimeType normalizes a MIME type by removing parameters
func (v *MimeValidator) normalizeMimeType(mimeType string) string {
	// Split by semicolon and take the first part
	parts := strings.Split(mimeType, ";")
	return strings.TrimSpace(parts[0])
}

// DetectMimeType detects MIME type from file content
func DetectMimeType(data []byte) string {
	return http.DetectContentType(data)
}

// DetectMimeTypeFromFilename returns MIME type based on file extension
func DetectMimeTypeFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

// GetMimeCategory returns the category of a MIME type
func GetMimeCategory(mimeType string) string {
	parts := strings.Split(mimeType, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// IsImageMime checks if MIME type is an image
func IsImageMime(mimeType string) bool {
	return GetMimeCategory(mimeType) == "image"
}

// IsVideoMime checks if MIME type is a video
func IsVideoMime(mimeType string) bool {
	return GetMimeCategory(mimeType) == "video"
}

// IsAudioMime checks if MIME type is audio
func IsAudioMime(mimeType string) bool {
	return GetMimeCategory(mimeType) == "audio"
}

// IsDocumentMime checks if MIME type is a document
func IsDocumentMime(mimeType string) bool {
	docTypes := map[string]bool{
		"application/pdf":    true,
		"application/msword": true,
		"text/plain":         true,
		"text/csv":           true,
		"text/html":          true,
		"text/markdown":      true,
		"application/json":   true,
		"application/xml":    true,
	}
	// Check for OpenXML formats
	if strings.Contains(mimeType, "openxmlformats") {
		return true
	}
	return docTypes[mimeType]
}

// AllowedMimeTypesList returns a list of all allowed MIME types
func (v *MimeValidator) AllowedMimeTypesList() []string {
	types := make([]string, 0, len(v.allowedTypes))
	for t := range v.allowedTypes {
		types = append(types, t)
	}
	return types
}

// AddAllowedType adds a MIME type to the allowed list
func (v *MimeValidator) AddAllowedType(mimeType string) {
	v.allowedTypes[mimeType] = true
}

// RemoveAllowedType removes a MIME type from the allowed list
func (v *MimeValidator) RemoveAllowedType(mimeType string) {
	delete(v.allowedTypes, mimeType)
}
