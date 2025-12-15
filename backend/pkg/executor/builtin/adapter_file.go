package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// FileToBytesExecutor reads file from storage as bytes
type FileToBytesExecutor struct {
	*executor.BaseExecutor
	manager filestorage.Manager
}

// NewFileToBytesExecutor creates a new file to bytes executor
func NewFileToBytesExecutor(manager filestorage.Manager) *FileToBytesExecutor {
	return &FileToBytesExecutor{
		BaseExecutor: executor.NewBaseExecutor("file_to_bytes"),
		manager:      manager,
	}
}

// Execute reads file from storage
//
// Config:
//   - storage_id: storage ID (default: "default")
//   - file_id: file ID to read (supports templates)
//   - output_format: "raw" | "base64" (default: "base64")
//
// Input: file ID (string) or map with "file_id" field
//
// Output:
//   - success: true
//   - result: file content (bytes or base64 string)
//   - file_id: file ID
//   - file_name: original file name
//   - mime_type: file MIME type
//   - size: file size in bytes
//   - format: output format used
//   - duration_ms: execution time
func (e *FileToBytesExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	startTime := time.Now()

	// Get configuration
	storageID := e.GetStringDefault(config, "storage_id", "default")
	outputFormat := e.GetStringDefault(config, "output_format", "base64")

	// Extract file ID from config or input
	fileID, err := e.GetString(config, "file_id")
	if err != nil {
		// Try to get from input
		fileID, err = e.extractFileID(input)
		if err != nil {
			return nil, fmt.Errorf("file_to_bytes: %w", err)
		}
	}

	// Get storage
	storage, err := e.manager.GetStorage(storageID)
	if err != nil {
		return nil, fmt.Errorf("file_to_bytes: failed to get storage: %w", err)
	}

	// Get file entry and content
	entry, reader, err := storage.Get(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("file_to_bytes: failed to read file: %w", err)
	}
	defer reader.Close()

	// Read all bytes
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("file_to_bytes: failed to read file content: %w", err)
	}

	// Format output
	var result interface{}
	switch outputFormat {
	case "base64":
		result = base64.StdEncoding.EncodeToString(content)
	case "raw":
		result = content
	default:
		return nil, fmt.Errorf("file_to_bytes: invalid output_format: %s", outputFormat)
	}

	return map[string]interface{}{
		"success":     true,
		"result":      result,
		"file_id":     entry.ID,
		"file_name":   entry.Name,
		"mime_type":   entry.MimeType,
		"size":        entry.Size,
		"format":      outputFormat,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}, nil
}

// Validate validates the configuration
func (e *FileToBytesExecutor) Validate(config map[string]interface{}) error {
	// file_id is required
	if _, err := e.GetString(config, "file_id"); err != nil {
		return fmt.Errorf("file_id is required")
	}

	// Output format validation
	outputFormat := e.GetStringDefault(config, "output_format", "base64")
	validFormats := map[string]bool{
		"raw":    true,
		"base64": true,
	}
	if !validFormats[outputFormat] {
		return fmt.Errorf("invalid output_format: %s (must be: raw, base64)", outputFormat)
	}

	return nil
}

// extractFileID extracts file ID from input
func (e *FileToBytesExecutor) extractFileID(input interface{}) (string, error) {
	switch v := input.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		if fileID, ok := v["file_id"].(string); ok {
			return fileID, nil
		}
		return "", fmt.Errorf("expected 'file_id' field in input map")
	default:
		return "", fmt.Errorf("unsupported input type: %T (expected string or map)", input)
	}
}

// BytesToFileExecutor saves bytes to file storage
type BytesToFileExecutor struct {
	*executor.BaseExecutor
	manager filestorage.Manager
}

// NewBytesToFileExecutor creates a new bytes to file executor
func NewBytesToFileExecutor(manager filestorage.Manager) *BytesToFileExecutor {
	return &BytesToFileExecutor{
		BaseExecutor: executor.NewBaseExecutor("bytes_to_file"),
		manager:      manager,
	}
}

// Execute saves bytes to file storage
//
// Config:
//   - storage_id: storage ID (default: "default")
//   - file_name: file name (supports templates)
//   - mime_type: MIME type (auto-detect if empty)
//   - access_scope: "workflow" | "edge" | "result" (default: "workflow")
//   - ttl: TTL in seconds (0 = no expiration) (default: 0)
//   - tags: array of tags (default: [])
//
// Input: bytes ([]byte, string, or map with "data" field)
//
// Output:
//   - success: true
//   - file_id: stored file ID
//   - storage_id: storage ID
//   - file_name: file name
//   - mime_type: detected/configured MIME type
//   - size: file size in bytes
//   - checksum: file checksum
//   - access_scope: access scope
//   - duration_ms: execution time
func (e *BytesToFileExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	startTime := time.Now()

	// Get configuration
	storageID := e.GetStringDefault(config, "storage_id", "default")
	fileName, err := e.GetString(config, "file_name")
	if err != nil {
		return nil, fmt.Errorf("bytes_to_file: file_name is required: %w", err)
	}

	mimeType := e.GetStringDefault(config, "mime_type", "")
	accessScope := e.GetStringDefault(config, "access_scope", "workflow")
	ttl := e.GetIntDefault(config, "ttl", 0)

	// Get tags
	var tags []string
	if tagsRaw, ok := config["tags"].([]interface{}); ok {
		for _, tag := range tagsRaw {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// Extract bytes from input
	data, err := e.extractBytes(input)
	if err != nil {
		return nil, fmt.Errorf("bytes_to_file: %w", err)
	}

	// Get storage
	storage, err := e.manager.GetStorage(storageID)
	if err != nil {
		return nil, fmt.Errorf("bytes_to_file: failed to get storage: %w", err)
	}

	// Auto-detect MIME type if not provided
	if mimeType == "" {
		// Detect from content first, fallback to filename
		mimeType = filestorage.DetectMimeType(data)
		if mimeType == "application/octet-stream" {
			mimeType = filestorage.DetectMimeTypeFromFilename(fileName)
		}
	}

	// Validate access scope
	if !models.AccessScope(accessScope).IsValid() {
		return nil, fmt.Errorf("bytes_to_file: invalid access_scope: %s", accessScope)
	}

	// Create file entry
	entry := &models.FileEntry{
		StorageID:   storageID,
		Name:        fileName,
		MimeType:    mimeType,
		Size:        int64(len(data)),
		AccessScope: models.AccessScope(accessScope),
		Tags:        tags,
		Metadata:    make(map[string]interface{}),
	}

	// Set TTL if provided
	if ttl > 0 {
		entry.SetTTL(time.Duration(ttl) * time.Second)
	}

	// Store file
	stored, err := storage.Store(ctx, entry, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("bytes_to_file: failed to store file: %w", err)
	}

	return map[string]interface{}{
		"success":      true,
		"file_id":      stored.ID,
		"storage_id":   stored.StorageID,
		"file_name":    stored.Name,
		"mime_type":    stored.MimeType,
		"size":         stored.Size,
		"checksum":     stored.Checksum,
		"access_scope": accessScope,
		"duration_ms":  time.Since(startTime).Milliseconds(),
	}, nil
}

// Validate validates the configuration
func (e *BytesToFileExecutor) Validate(config map[string]interface{}) error {
	// file_name is required
	if _, err := e.GetString(config, "file_name"); err != nil {
		return fmt.Errorf("file_name is required")
	}

	// Access scope validation
	accessScope := e.GetStringDefault(config, "access_scope", "workflow")
	validScopes := map[string]bool{
		"workflow": true,
		"edge":     true,
		"result":   true,
	}
	if !validScopes[accessScope] {
		return fmt.Errorf("invalid access_scope: %s (must be: workflow, edge, result)", accessScope)
	}

	// TTL validation
	ttl := e.GetIntDefault(config, "ttl", 0)
	if ttl < 0 {
		return fmt.Errorf("ttl must be >= 0")
	}

	return nil
}

// extractBytes extracts bytes from various input types
func (e *BytesToFileExecutor) extractBytes(input interface{}) ([]byte, error) {
	switch v := input.(type) {
	case []byte:
		return v, nil
	case string:
		// Try to decode as base64 first
		if decoded, err := base64.StdEncoding.DecodeString(v); err == nil {
			// Check if it looks like base64
			if len(v) > 0 && len(v)%4 == 0 {
				return decoded, nil
			}
		}
		// Use string as UTF-8 bytes
		return []byte(v), nil
	case map[string]interface{}:
		// Try to extract from "data" field
		if data, ok := v["data"]; ok {
			return e.extractBytes(data)
		}
		return nil, fmt.Errorf("expected 'data' field in input map")
	default:
		return nil, fmt.Errorf("unsupported input type: %T (expected []byte, string, or map)", input)
	}
}
