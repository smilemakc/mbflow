package filestorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/pkg/models"
)

// LocalProvider implements Provider for local disk storage
type LocalProvider struct {
	basePath string
	mu       sync.RWMutex
}

// NewLocalProvider creates a new local storage provider
func NewLocalProvider(basePath string) (*LocalProvider, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalProvider{
		basePath: basePath,
	}, nil
}

// Type returns the storage type
func (p *LocalProvider) Type() models.StorageType {
	return models.StorageTypeLocal
}

// Store stores a file to local disk
func (p *LocalProvider) Store(ctx context.Context, entry *models.FileEntry, reader io.Reader) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Generate unique path if not provided
	relativePath := entry.Path
	if relativePath == "" {
		// Create path: storageID/year/month/uuid/filename
		relativePath = p.generatePath(entry)
	}

	fullPath := filepath.Join(p.basePath, relativePath)

	// Create directory structure
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content and calculate checksum
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	size, err := io.Copy(writer, reader)
	if err != nil {
		// Clean up on error
		os.Remove(fullPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Update entry with calculated values
	entry.Size = size
	entry.Checksum = hex.EncodeToString(hasher.Sum(nil))
	entry.Path = relativePath

	return relativePath, nil
}

// generatePath generates a unique file path
func (p *LocalProvider) generatePath(entry *models.FileEntry) string {
	// Sanitize filename
	safeName := sanitizeFilename(entry.Name)
	if safeName == "" {
		safeName = "file"
	}

	// Generate unique ID
	uniqueID := uuid.New().String()[:8]

	// Build path: storageID/uniqueID/filename
	return filepath.Join(entry.StorageID, uniqueID, safeName)
}

// sanitizeFilename removes unsafe characters from filename
func sanitizeFilename(name string) string {
	// Replace path separators and other unsafe characters
	unsafe := []string{"/", "\\", "..", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Limit length
	if len(result) > 200 {
		result = result[:200]
	}
	return result
}

// Get retrieves a file from local disk
func (p *LocalProvider) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	fullPath := filepath.Join(p.basePath, path)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file from local disk
func (p *LocalProvider) Delete(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	fullPath := filepath.Join(p.basePath, path)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to delete
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Try to clean up empty parent directories
	p.cleanupEmptyDirs(filepath.Dir(fullPath))

	return nil
}

// cleanupEmptyDirs removes empty parent directories up to basePath
func (p *LocalProvider) cleanupEmptyDirs(dir string) {
	for dir != p.basePath && strings.HasPrefix(dir, p.basePath) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// Exists checks if a file exists
func (p *LocalProvider) Exists(ctx context.Context, path string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	fullPath := filepath.Join(p.basePath, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetUsage returns storage usage statistics
func (p *LocalProvider) GetUsage(ctx context.Context) (*models.StorageUsage, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var totalSize int64
	var fileCount int64

	err := filepath.Walk(p.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate usage: %w", err)
	}

	return &models.StorageUsage{
		TotalSize: totalSize,
		FileCount: fileCount,
	}, nil
}

// Close closes the provider
func (p *LocalProvider) Close() error {
	return nil
}

// LocalProviderFactory creates local storage providers
type LocalProviderFactory struct{}

// NewLocalProviderFactory creates a new local provider factory
func NewLocalProviderFactory() *LocalProviderFactory {
	return &LocalProviderFactory{}
}

// Type returns the storage type
func (f *LocalProviderFactory) Type() models.StorageType {
	return models.StorageTypeLocal
}

// Create creates a new local provider
func (f *LocalProviderFactory) Create(config *models.StorageConfig) (Provider, error) {
	if config.BasePath == "" {
		return nil, fmt.Errorf("base_path is required for local storage")
	}
	return NewLocalProvider(config.BasePath)
}
