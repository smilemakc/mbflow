package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== AccessScope Tests ====================

func TestAccessScope_Constants(t *testing.T) {
	assert.Equal(t, AccessScope("workflow"), ScopeWorkflow)
	assert.Equal(t, AccessScope("edge"), ScopeEdge)
	assert.Equal(t, AccessScope("result"), ScopeResult)
}

func TestAccessScope_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		scope    AccessScope
		expected bool
	}{
		{"workflow scope valid", ScopeWorkflow, true},
		{"edge scope valid", ScopeEdge, true},
		{"result scope valid", ScopeResult, true},
		{"invalid scope", AccessScope("invalid"), false},
		{"empty scope", AccessScope(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.scope.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== FileEntry.Validate Tests ====================

func TestFileEntry_Validate_Success(t *testing.T) {
	entry := &FileEntry{
		ID:          "file_123",
		StorageID:   "storage_1",
		Name:        "document.pdf",
		MimeType:    "application/pdf",
		Size:        1024,
		AccessScope: ScopeWorkflow,
	}

	err := entry.Validate()
	assert.NoError(t, err)
}

func TestFileEntry_Validate_MissingID(t *testing.T) {
	entry := &FileEntry{
		StorageID:   "storage_1",
		Name:        "document.pdf",
		MimeType:    "application/pdf",
		AccessScope: ScopeWorkflow,
	}

	err := entry.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file ID is required")
}

func TestFileEntry_Validate_MissingStorageID(t *testing.T) {
	entry := &FileEntry{
		ID:          "file_123",
		Name:        "document.pdf",
		MimeType:    "application/pdf",
		AccessScope: ScopeWorkflow,
	}

	err := entry.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage ID is required")
}

func TestFileEntry_Validate_MissingName(t *testing.T) {
	entry := &FileEntry{
		ID:          "file_123",
		StorageID:   "storage_1",
		MimeType:    "application/pdf",
		AccessScope: ScopeWorkflow,
	}

	err := entry.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file name is required")
}

func TestFileEntry_Validate_MissingMimeType(t *testing.T) {
	entry := &FileEntry{
		ID:          "file_123",
		StorageID:   "storage_1",
		Name:        "document.pdf",
		AccessScope: ScopeWorkflow,
	}

	err := entry.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MIME type is required")
}

func TestFileEntry_Validate_InvalidAccessScope(t *testing.T) {
	entry := &FileEntry{
		ID:          "file_123",
		StorageID:   "storage_1",
		Name:        "document.pdf",
		MimeType:    "application/pdf",
		AccessScope: AccessScope("invalid"),
	}

	err := entry.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid access scope")
}

func TestFileEntry_Validate_NegativeSize(t *testing.T) {
	entry := &FileEntry{
		ID:          "file_123",
		StorageID:   "storage_1",
		Name:        "document.pdf",
		MimeType:    "application/pdf",
		Size:        -100,
		AccessScope: ScopeWorkflow,
	}

	err := entry.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file size cannot be negative")
}

func TestFileEntry_Validate_ZeroSize(t *testing.T) {
	entry := &FileEntry{
		ID:          "file_123",
		StorageID:   "storage_1",
		Name:        "empty.txt",
		MimeType:    "text/plain",
		Size:        0, // Zero size is valid (empty file)
		AccessScope: ScopeWorkflow,
	}

	err := entry.Validate()
	assert.NoError(t, err)
}

// ==================== FileEntry.IsExpired Tests ====================

func TestFileEntry_IsExpired_NoExpiration(t *testing.T) {
	entry := &FileEntry{
		ExpiresAt: nil,
	}

	result := entry.IsExpired()
	assert.False(t, result)
}

func TestFileEntry_IsExpired_NotExpired(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	entry := &FileEntry{
		ExpiresAt: &future,
	}

	result := entry.IsExpired()
	assert.False(t, result)
}

func TestFileEntry_IsExpired_Expired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	entry := &FileEntry{
		ExpiresAt: &past,
	}

	result := entry.IsExpired()
	assert.True(t, result)
}

func TestFileEntry_IsExpired_JustExpired(t *testing.T) {
	// Just passed expiration (1 second ago)
	justPast := time.Now().Add(-1 * time.Second)
	entry := &FileEntry{
		ExpiresAt: &justPast,
	}

	result := entry.IsExpired()
	assert.True(t, result)
}

// ==================== FileEntry.SetTTL Tests ====================

func TestFileEntry_SetTTL_Success(t *testing.T) {
	entry := &FileEntry{}
	ttl := 1 * time.Hour

	beforeSet := time.Now()
	entry.SetTTL(ttl)
	afterSet := time.Now()

	require.NotNil(t, entry.TTL)
	require.NotNil(t, entry.ExpiresAt)

	assert.Equal(t, ttl, *entry.TTL)

	// ExpiresAt should be approximately Now() + TTL
	expectedExpiry := beforeSet.Add(ttl)
	maxExpectedExpiry := afterSet.Add(ttl)

	assert.True(t, entry.ExpiresAt.After(expectedExpiry) || entry.ExpiresAt.Equal(expectedExpiry))
	assert.True(t, entry.ExpiresAt.Before(maxExpectedExpiry) || entry.ExpiresAt.Equal(maxExpectedExpiry))
}

func TestFileEntry_SetTTL_ShortDuration(t *testing.T) {
	entry := &FileEntry{}
	ttl := 5 * time.Minute

	entry.SetTTL(ttl)

	require.NotNil(t, entry.TTL)
	require.NotNil(t, entry.ExpiresAt)
	assert.Equal(t, ttl, *entry.TTL)

	// Should not be expired immediately
	assert.False(t, entry.IsExpired())
}

func TestFileEntry_SetTTL_LongDuration(t *testing.T) {
	entry := &FileEntry{}
	ttl := 24 * time.Hour

	entry.SetTTL(ttl)

	require.NotNil(t, entry.TTL)
	require.NotNil(t, entry.ExpiresAt)
	assert.Equal(t, ttl, *entry.TTL)
}

// ==================== FileEntry JSON Tests ====================

func TestFileEntry_JSONMarshaling(t *testing.T) {
	now := time.Now()
	ttl := 1 * time.Hour
	workflowID := "wf_123"
	executionID := "exec_456"
	sourceNodeID := "node_789"

	entry := &FileEntry{
		ID:           "file_123",
		StorageID:    "storage_1",
		Name:         "document.pdf",
		Path:         "/files/document.pdf",
		MimeType:     "application/pdf",
		Size:         1024,
		Checksum:     "abc123",
		AccessScope:  ScopeWorkflow,
		Tags:         []string{"important", "archived"},
		Metadata:     map[string]any{"author": "John Doe"},
		TTL:          &ttl,
		ExpiresAt:    &now,
		WorkflowID:   &workflowID,
		ExecutionID:  &executionID,
		SourceNodeID: &sourceNodeID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var unmarshaled FileEntry
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, entry.ID, unmarshaled.ID)
	assert.Equal(t, entry.Name, unmarshaled.Name)
	assert.Equal(t, entry.MimeType, unmarshaled.MimeType)
	assert.Equal(t, entry.AccessScope, unmarshaled.AccessScope)
	assert.Len(t, unmarshaled.Tags, 2)
	assert.NotNil(t, unmarshaled.Metadata)
}

// ==================== StorageType Tests ====================

func TestStorageType_Constants(t *testing.T) {
	assert.Equal(t, StorageType("local"), StorageTypeLocal)
	assert.Equal(t, StorageType("s3"), StorageTypeS3)
}

// ==================== StorageConfig Tests ====================

func TestStorageConfig_JSONMarshaling(t *testing.T) {
	ttl := 24 * time.Hour
	config := &StorageConfig{
		Type:        StorageTypeLocal,
		BasePath:    "/var/lib/mbflow/storage",
		MaxSize:     1024 * 1024 * 1024, // 1GB
		MaxFileSize: 100 * 1024 * 1024,  // 100MB
		DefaultTTL:  &ttl,
		Options: map[string]any{
			"encryption":  true,
			"compression": "gzip",
		},
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled StorageConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.Type, unmarshaled.Type)
	assert.Equal(t, config.BasePath, unmarshaled.BasePath)
	assert.Equal(t, config.MaxSize, unmarshaled.MaxSize)
	assert.Equal(t, config.MaxFileSize, unmarshaled.MaxFileSize)
	require.NotNil(t, unmarshaled.DefaultTTL)
	assert.Equal(t, ttl, *unmarshaled.DefaultTTL)
	assert.NotNil(t, unmarshaled.Options)
}

func TestStorageConfig_S3Type(t *testing.T) {
	config := &StorageConfig{
		Type: StorageTypeS3,
		Options: map[string]any{
			"bucket":  "my-bucket",
			"region":  "us-east-1",
			"api_key": "secret",
		},
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled StorageConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, StorageTypeS3, unmarshaled.Type)
	assert.Equal(t, "my-bucket", unmarshaled.Options["bucket"])
}

// ==================== StorageUsage Tests ====================

func TestStorageUsage_JSONMarshaling(t *testing.T) {
	usage := &StorageUsage{
		StorageID:    "storage_1",
		TotalSize:    524288000, // 500MB
		FileCount:    150,
		MaxSize:      1073741824, // 1GB
		UsagePercent: 48.83,
	}

	data, err := json.Marshal(usage)
	require.NoError(t, err)

	var unmarshaled StorageUsage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, usage.StorageID, unmarshaled.StorageID)
	assert.Equal(t, usage.TotalSize, unmarshaled.TotalSize)
	assert.Equal(t, usage.FileCount, unmarshaled.FileCount)
	assert.Equal(t, usage.MaxSize, unmarshaled.MaxSize)
	assert.InDelta(t, usage.UsagePercent, unmarshaled.UsagePercent, 0.01)
}

func TestStorageUsage_ZeroMaxSize(t *testing.T) {
	usage := &StorageUsage{
		StorageID:    "storage_unlimited",
		TotalSize:    524288000,
		FileCount:    150,
		MaxSize:      0, // Unlimited
		UsagePercent: 0,
	}

	data, err := json.Marshal(usage)
	require.NoError(t, err)

	var unmarshaled StorageUsage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, int64(0), unmarshaled.MaxSize)
}

// ==================== IsMimeTypeAllowed Tests ====================

func TestIsMimeTypeAllowed_Images(t *testing.T) {
	tests := []struct {
		mimeType string
		allowed  bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/svg+xml", true},
		{"image/bmp", true},
		{"image/tiff", true},
		{"image/x-icon", false}, // Not in allowed list
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := IsMimeTypeAllowed(tt.mimeType)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

func TestIsMimeTypeAllowed_Documents(t *testing.T) {
	tests := []struct {
		mimeType string
		allowed  bool
	}{
		{"application/pdf", true},
		{"application/msword", true},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", true}, // .docx
		{"application/vnd.ms-excel", true},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", true}, // .xlsx
		{"application/rtf", false},                                                  // Not in allowed list
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := IsMimeTypeAllowed(tt.mimeType)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

func TestIsMimeTypeAllowed_Audio(t *testing.T) {
	tests := []struct {
		mimeType string
		allowed  bool
	}{
		{"audio/mpeg", true},
		{"audio/wav", true},
		{"audio/ogg", true},
		{"audio/webm", true},
		{"audio/flac", true},
		{"audio/aac", false}, // Not in allowed list
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := IsMimeTypeAllowed(tt.mimeType)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

func TestIsMimeTypeAllowed_Video(t *testing.T) {
	tests := []struct {
		mimeType string
		allowed  bool
	}{
		{"video/mp4", true},
		{"video/webm", true},
		{"video/ogg", true},
		{"video/mpeg", true},
		{"video/quicktime", true},
		{"video/x-matroska", false}, // Not in allowed list
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := IsMimeTypeAllowed(tt.mimeType)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

func TestIsMimeTypeAllowed_Text(t *testing.T) {
	tests := []struct {
		mimeType string
		allowed  bool
	}{
		{"text/plain", true},
		{"text/csv", true},
		{"text/html", true},
		{"text/markdown", true},
		{"application/json", true},
		{"application/xml", true},
		{"text/yaml", false}, // Not in allowed list
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := IsMimeTypeAllowed(tt.mimeType)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

func TestIsMimeTypeAllowed_Archives(t *testing.T) {
	tests := []struct {
		mimeType string
		allowed  bool
	}{
		{"application/zip", true},
		{"application/gzip", true},
		{"application/x-tar", true},
		{"application/x-rar-compressed", true},
		{"application/x-7z-compressed", true},
		{"application/x-bzip2", false}, // Not in allowed list
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := IsMimeTypeAllowed(tt.mimeType)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

func TestIsMimeTypeAllowed_Invalid(t *testing.T) {
	tests := []string{
		"invalid/mime",
		"application/x-executable",
		"application/x-sh",
		"",
	}

	for _, mimeType := range tests {
		t.Run(mimeType, func(t *testing.T) {
			result := IsMimeTypeAllowed(mimeType)
			assert.False(t, result)
		})
	}
}

// ==================== Complex Integration Tests ====================

func TestFileEntry_CompleteLifecycle(t *testing.T) {
	// 1. Create file entry
	workflowID := "wf_123"
	entry := &FileEntry{
		ID:          "file_123",
		StorageID:   "storage_1",
		Name:        "document.pdf",
		Path:        "/files/wf_123/document.pdf",
		MimeType:    "application/pdf",
		Size:        1024 * 1024, // 1MB
		Checksum:    "sha256:abc123",
		AccessScope: ScopeWorkflow,
		Tags:        []string{"important"},
		WorkflowID:  &workflowID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 2. Validate
	err := entry.Validate()
	require.NoError(t, err)

	// 3. Set TTL
	entry.SetTTL(1 * time.Hour)
	assert.NotNil(t, entry.TTL)
	assert.NotNil(t, entry.ExpiresAt)

	// 4. Check not expired
	assert.False(t, entry.IsExpired())

	// 5. Marshal to JSON
	data, err := json.Marshal(entry)
	require.NoError(t, err)

	// 6. Unmarshal
	var unmarshaled FileEntry
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// 7. Validate unmarshaled
	err = unmarshaled.Validate()
	require.NoError(t, err)

	// 8. Verify data integrity
	assert.Equal(t, entry.ID, unmarshaled.ID)
	assert.Equal(t, entry.Name, unmarshaled.Name)
	assert.Equal(t, entry.AccessScope, unmarshaled.AccessScope)
}

func TestFileStorage_MultipleScopes(t *testing.T) {
	scopes := []AccessScope{ScopeWorkflow, ScopeEdge, ScopeResult}

	for _, scope := range scopes {
		t.Run(string(scope), func(t *testing.T) {
			entry := &FileEntry{
				ID:          "file_" + string(scope),
				StorageID:   "storage_1",
				Name:        "test.txt",
				MimeType:    "text/plain",
				Size:        100,
				AccessScope: scope,
			}

			err := entry.Validate()
			assert.NoError(t, err)
			assert.True(t, scope.IsValid())
		})
	}
}
