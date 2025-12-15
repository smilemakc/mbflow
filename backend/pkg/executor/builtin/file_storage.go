package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// FileStorageExecutor executes file storage operations
type FileStorageExecutor struct {
	*executor.BaseExecutor
	manager filestorage.Manager
}

// NewFileStorageExecutor creates a new file storage executor
func NewFileStorageExecutor(manager filestorage.Manager) *FileStorageExecutor {
	return &FileStorageExecutor{
		BaseExecutor: executor.NewBaseExecutor("file_storage"),
		manager:      manager,
	}
}

// Execute executes a file storage operation
//
// Config:
//   - action: "store" | "get" | "delete" | "list" | "metadata"
//   - storage_id: Storage ID (optional, defaults to workflow-{workflow_id})
//   - file_data: Base64 encoded file data (for store)
//   - file_url: URL to download file from (for store)
//   - file_name: File name
//   - mime_type: MIME type (auto-detected if not provided)
//   - file_id: File ID (for get/delete/metadata)
//   - access_scope: "workflow" | "edge" | "result" (default: workflow)
//   - tags: Array of tags
//   - ttl: TTL in seconds (0 = no expiration)
//
// Output:
//   - file_id: Stored/retrieved file ID
//   - file_url: URL to access the file
//   - file_data: Base64 encoded file content (for get)
//   - metadata: File metadata
//   - files: Array of files (for list)
func (e *FileStorageExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	startTime := time.Now()

	action, err := e.GetString(config, "action")
	if err != nil {
		return nil, fmt.Errorf("action is required: %w", err)
	}

	var result interface{}
	switch action {
	case "store":
		result, err = e.executeStore(ctx, config, input)
	case "get":
		result, err = e.executeGet(ctx, config)
	case "delete":
		result, err = e.executeDelete(ctx, config)
	case "list":
		result, err = e.executeList(ctx, config)
	case "metadata":
		result, err = e.executeMetadata(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}

	if err != nil {
		return nil, fmt.Errorf("file storage %s failed: %w", action, err)
	}

	// Add duration to result
	if resultMap, ok := result.(map[string]interface{}); ok {
		resultMap["duration_ms"] = time.Since(startTime).Milliseconds()
		resultMap["action"] = action
		return resultMap, nil
	}

	return result, nil
}

// executeStore stores a file
func (e *FileStorageExecutor) executeStore(ctx context.Context, config map[string]interface{}, input interface{}) (map[string]interface{}, error) {
	// Get storage
	storageID := e.GetStringDefault(config, "storage_id", "default")
	storage, err := e.manager.GetStorage(storageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	// Get file data
	var reader io.Reader
	var fileName string
	var mimeType string
	var size int64

	// Check for base64 data
	if fileData := e.GetStringDefault(config, "file_data", ""); fileData != "" {
		decoded, err := base64.StdEncoding.DecodeString(fileData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 file data: %w", err)
		}
		reader = bytes.NewReader(decoded)
		size = int64(len(decoded))

		// Auto-detect MIME type from content
		if mimeType == "" {
			mimeType = filestorage.DetectMimeType(decoded[:min(512, len(decoded))])
		}
	} else if fileURL := e.GetStringDefault(config, "file_url", ""); fileURL != "" {
		// Download from URL
		resp, err := http.Get(fileURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download file from URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
		}

		// Read into buffer
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read file from URL: %w", err)
		}
		reader = bytes.NewReader(data)
		size = int64(len(data))

		// Get MIME from response or detect
		mimeType = resp.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = filestorage.DetectMimeType(data[:min(512, len(data))])
		}

		// Extract filename from URL if not provided
		if fileName == "" {
			parts := strings.Split(fileURL, "/")
			if len(parts) > 0 {
				fileName = parts[len(parts)-1]
			}
		}
	} else {
		return nil, fmt.Errorf("either file_data or file_url is required for store action")
	}

	// Get file name
	fileName = e.GetStringDefault(config, "file_name", fileName)
	if fileName == "" {
		fileName = fmt.Sprintf("file_%s", uuid.New().String()[:8])
	}

	// Get MIME type (override if provided)
	if configMime := e.GetStringDefault(config, "mime_type", ""); configMime != "" {
		mimeType = configMime
	}
	if mimeType == "" {
		mimeType = filestorage.DetectMimeTypeFromFilename(fileName)
	}

	// Get access scope
	accessScope := e.GetStringDefault(config, "access_scope", "workflow")
	if !models.AccessScope(accessScope).IsValid() {
		return nil, fmt.Errorf("invalid access_scope: %s", accessScope)
	}

	// Get tags
	var tags []string
	if tagsVal, ok := config["tags"]; ok {
		if tagsArr, ok := tagsVal.([]interface{}); ok {
			for _, t := range tagsArr {
				if str, ok := t.(string); ok {
					tags = append(tags, str)
				}
			}
		}
	}

	// Get TTL
	ttl := e.GetIntDefault(config, "ttl", 0)

	// Create file entry
	entry := &models.FileEntry{
		StorageID:   storageID,
		Name:        fileName,
		MimeType:    mimeType,
		Size:        size,
		AccessScope: models.AccessScope(accessScope),
		Tags:        tags,
		Metadata:    make(map[string]interface{}),
	}

	// Set TTL if provided
	if ttl > 0 {
		entry.SetTTL(time.Duration(ttl) * time.Second)
	}

	// Store file
	stored, err := storage.Store(ctx, entry, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	return map[string]interface{}{
		"success":      true,
		"file_id":      stored.ID,
		"storage_id":   stored.StorageID,
		"file_name":    stored.Name,
		"mime_type":    stored.MimeType,
		"size":         stored.Size,
		"checksum":     stored.Checksum,
		"access_scope": stored.AccessScope,
		"expires_at":   stored.ExpiresAt,
	}, nil
}

// executeGet retrieves a file
func (e *FileStorageExecutor) executeGet(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error) {
	storageID := e.GetStringDefault(config, "storage_id", "default")
	storage, err := e.manager.GetStorage(storageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	fileID, err := e.GetString(config, "file_id")
	if err != nil {
		return nil, fmt.Errorf("file_id is required for get action: %w", err)
	}

	entry, reader, err := storage.Get(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	defer reader.Close()

	// Read file content
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return map[string]interface{}{
		"success":      true,
		"file_id":      entry.ID,
		"storage_id":   entry.StorageID,
		"file_name":    entry.Name,
		"mime_type":    entry.MimeType,
		"size":         entry.Size,
		"file_data":    base64.StdEncoding.EncodeToString(data),
		"checksum":     entry.Checksum,
		"access_scope": entry.AccessScope,
		"tags":         entry.Tags,
		"metadata":     entry.Metadata,
	}, nil
}

// executeDelete deletes a file
func (e *FileStorageExecutor) executeDelete(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error) {
	storageID := e.GetStringDefault(config, "storage_id", "default")
	storage, err := e.manager.GetStorage(storageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	fileID, err := e.GetString(config, "file_id")
	if err != nil {
		return nil, fmt.Errorf("file_id is required for delete action: %w", err)
	}

	if err := storage.Delete(ctx, fileID); err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}

	return map[string]interface{}{
		"success":    true,
		"file_id":    fileID,
		"storage_id": storageID,
	}, nil
}

// executeList lists files
func (e *FileStorageExecutor) executeList(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error) {
	storageID := e.GetStringDefault(config, "storage_id", "default")
	storage, err := e.manager.GetStorage(storageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	query := &filestorage.FileQuery{
		StorageID: storageID,
		Limit:     e.GetIntDefault(config, "limit", 100),
		Offset:    e.GetIntDefault(config, "offset", 0),
	}

	// Add filters
	if accessScope := e.GetStringDefault(config, "access_scope", ""); accessScope != "" {
		query.AccessScope = models.AccessScope(accessScope)
	}

	// Get tags
	if tagsVal, ok := config["tags"]; ok {
		if tagsArr, ok := tagsVal.([]interface{}); ok {
			for _, t := range tagsArr {
				if str, ok := t.(string); ok {
					query.Tags = append(query.Tags, str)
				}
			}
		}
	}

	files, err := storage.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Convert to output format
	fileList := make([]map[string]interface{}, len(files))
	for i, f := range files {
		fileList[i] = map[string]interface{}{
			"file_id":      f.ID,
			"storage_id":   f.StorageID,
			"file_name":    f.Name,
			"mime_type":    f.MimeType,
			"size":         f.Size,
			"access_scope": f.AccessScope,
			"tags":         f.Tags,
			"created_at":   f.CreatedAt,
		}
	}

	return map[string]interface{}{
		"success":    true,
		"storage_id": storageID,
		"files":      fileList,
		"count":      len(files),
	}, nil
}

// executeMetadata gets file metadata
func (e *FileStorageExecutor) executeMetadata(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error) {
	storageID := e.GetStringDefault(config, "storage_id", "default")
	storage, err := e.manager.GetStorage(storageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	fileID, err := e.GetString(config, "file_id")
	if err != nil {
		return nil, fmt.Errorf("file_id is required for metadata action: %w", err)
	}

	entry, err := storage.GetMetadata(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	return map[string]interface{}{
		"success":      true,
		"file_id":      entry.ID,
		"storage_id":   entry.StorageID,
		"file_name":    entry.Name,
		"mime_type":    entry.MimeType,
		"size":         entry.Size,
		"checksum":     entry.Checksum,
		"access_scope": entry.AccessScope,
		"tags":         entry.Tags,
		"metadata":     entry.Metadata,
		"expires_at":   entry.ExpiresAt,
		"created_at":   entry.CreatedAt,
	}, nil
}

// Validate validates the file storage executor configuration
func (e *FileStorageExecutor) Validate(config map[string]interface{}) error {
	// Validate action
	action, err := e.GetString(config, "action")
	if err != nil {
		return fmt.Errorf("action is required")
	}

	validActions := map[string]bool{
		"store": true, "get": true, "delete": true, "list": true, "metadata": true,
	}
	if !validActions[action] {
		return fmt.Errorf("invalid action: %s (must be: store, get, delete, list, metadata)", action)
	}

	// Validate action-specific requirements
	switch action {
	case "store":
		if config["file_data"] == nil && config["file_url"] == nil {
			return fmt.Errorf("either file_data or file_url is required for store action")
		}
	case "get", "delete", "metadata":
		if _, err := e.GetString(config, "file_id"); err != nil {
			return fmt.Errorf("file_id is required for %s action", action)
		}
	}

	// Validate access_scope if provided
	if accessScope := e.GetStringDefault(config, "access_scope", ""); accessScope != "" {
		if !models.AccessScope(accessScope).IsValid() {
			return fmt.Errorf("invalid access_scope: %s (must be: workflow, edge, result)", accessScope)
		}
	}

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
