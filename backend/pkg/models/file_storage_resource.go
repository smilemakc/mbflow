package models

import (
	"fmt"
	"time"
)

// FileStorageResource represents a file storage resource for a user
type FileStorageResource struct {
	BaseResource
	StorageLimitBytes int64  `json:"storage_limit_bytes"`
	UsedStorageBytes  int64  `json:"used_storage_bytes"`
	FileCount         int    `json:"file_count"`
	PricingPlanID     string `json:"pricing_plan_id,omitempty"`
}

// NewFileStorageResource creates a new file storage resource with default free tier limits
func NewFileStorageResource(ownerID, name string) *FileStorageResource {
	now := time.Now()
	return &FileStorageResource{
		BaseResource: BaseResource{
			Type:      ResourceTypeFileStorage,
			OwnerID:   ownerID,
			Name:      name,
			Status:    ResourceStatusActive,
			Metadata:  make(map[string]any),
			CreatedAt: now,
			UpdatedAt: now,
		},
		StorageLimitBytes: 5 * 1024 * 1024,
		UsedStorageBytes:  0,
		FileCount:         0,
	}
}

// Validate validates the file storage resource structure
func (f *FileStorageResource) Validate() error {
	if err := f.BaseResource.Validate(); err != nil {
		return err
	}
	if f.StorageLimitBytes <= 0 {
		return &ValidationError{Field: "storage_limit_bytes", Message: "storage limit must be positive"}
	}
	if f.UsedStorageBytes < 0 {
		return &ValidationError{Field: "used_storage_bytes", Message: "used storage cannot be negative"}
	}
	if f.UsedStorageBytes > f.StorageLimitBytes {
		return &ValidationError{Field: "used_storage_bytes", Message: "used storage exceeds limit"}
	}
	return nil
}

// GetUsagePercent returns the storage usage percentage
func (f *FileStorageResource) GetUsagePercent() float64 {
	if f.StorageLimitBytes == 0 {
		return 0
	}
	return float64(f.UsedStorageBytes) / float64(f.StorageLimitBytes) * 100
}

// CanAddFile checks if a file of the given size can be added
func (f *FileStorageResource) CanAddFile(fileSize int64) bool {
	return f.UsedStorageBytes+fileSize <= f.StorageLimitBytes
}

// AddFile increases counters after adding a file
func (f *FileStorageResource) AddFile(fileSize int64) error {
	if !f.CanAddFile(fileSize) {
		return fmt.Errorf("storage limit exceeded: %d + %d > %d",
			f.UsedStorageBytes, fileSize, f.StorageLimitBytes)
	}
	f.UsedStorageBytes += fileSize
	f.FileCount++
	f.UpdatedAt = time.Now()
	return nil
}

// RemoveFile decreases counters after removing a file
func (f *FileStorageResource) RemoveFile(fileSize int64) {
	f.UsedStorageBytes -= fileSize
	if f.UsedStorageBytes < 0 {
		f.UsedStorageBytes = 0
	}
	f.FileCount--
	if f.FileCount < 0 {
		f.FileCount = 0
	}
	f.UpdatedAt = time.Now()
}

// GetAvailableSpace returns the available storage space in bytes
func (f *FileStorageResource) GetAvailableSpace() int64 {
	return f.StorageLimitBytes - f.UsedStorageBytes
}

// UpdateLimit updates the storage limit
func (f *FileStorageResource) UpdateLimit(newLimitBytes int64) error {
	if newLimitBytes <= 0 {
		return &ValidationError{Field: "storage_limit_bytes", Message: "storage limit must be positive"}
	}
	if newLimitBytes < f.UsedStorageBytes {
		return &ValidationError{Field: "storage_limit_bytes", Message: "new limit cannot be less than used storage"}
	}
	f.StorageLimitBytes = newLimitBytes
	f.UpdatedAt = time.Now()
	return nil
}
