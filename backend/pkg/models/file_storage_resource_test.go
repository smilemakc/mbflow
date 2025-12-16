package models

import (
	"testing"
)

func TestNewFileStorageResource(t *testing.T) {
	resource := NewFileStorageResource("user-123", "My Storage")

	if resource.OwnerID != "user-123" {
		t.Errorf("OwnerID = %v, want %v", resource.OwnerID, "user-123")
	}
	if resource.Name != "My Storage" {
		t.Errorf("Name = %v, want %v", resource.Name, "My Storage")
	}
	if resource.Type != ResourceTypeFileStorage {
		t.Errorf("Type = %v, want %v", resource.Type, ResourceTypeFileStorage)
	}
	if resource.Status != ResourceStatusActive {
		t.Errorf("Status = %v, want %v", resource.Status, ResourceStatusActive)
	}
	if resource.StorageLimitBytes != 5*1024*1024 {
		t.Errorf("StorageLimitBytes = %v, want %v", resource.StorageLimitBytes, 5*1024*1024)
	}
	if resource.UsedStorageBytes != 0 {
		t.Errorf("UsedStorageBytes = %v, want %v", resource.UsedStorageBytes, 0)
	}
	if resource.FileCount != 0 {
		t.Errorf("FileCount = %v, want %v", resource.FileCount, 0)
	}
}

func TestFileStorageResource_Validate(t *testing.T) {
	tests := []struct {
		name     string
		resource *FileStorageResource
		wantErr  bool
	}{
		{
			name:     "valid resource",
			resource: NewFileStorageResource("user-123", "Test Storage"),
			wantErr:  false,
		},
		{
			name: "zero storage limit",
			resource: &FileStorageResource{
				BaseResource: BaseResource{
					OwnerID: "user-123",
					Name:    "Test Storage",
				},
				StorageLimitBytes: 0,
			},
			wantErr: true,
		},
		{
			name: "negative used storage",
			resource: &FileStorageResource{
				BaseResource: BaseResource{
					OwnerID: "user-123",
					Name:    "Test Storage",
				},
				StorageLimitBytes: 1000,
				UsedStorageBytes:  -100,
			},
			wantErr: true,
		},
		{
			name: "used exceeds limit",
			resource: &FileStorageResource{
				BaseResource: BaseResource{
					OwnerID: "user-123",
					Name:    "Test Storage",
				},
				StorageLimitBytes: 1000,
				UsedStorageBytes:  1500,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resource.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FileStorageResource.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileStorageResource_GetUsagePercent(t *testing.T) {
	tests := []struct {
		name            string
		limitBytes      int64
		usedBytes       int64
		expectedPercent float64
	}{
		{
			name:            "empty storage",
			limitBytes:      1000,
			usedBytes:       0,
			expectedPercent: 0,
		},
		{
			name:            "half full",
			limitBytes:      1000,
			usedBytes:       500,
			expectedPercent: 50,
		},
		{
			name:            "full storage",
			limitBytes:      1000,
			usedBytes:       1000,
			expectedPercent: 100,
		},
		{
			name:            "zero limit",
			limitBytes:      0,
			usedBytes:       0,
			expectedPercent: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewFileStorageResource("user-123", "Test Storage")
			resource.StorageLimitBytes = tt.limitBytes
			resource.UsedStorageBytes = tt.usedBytes

			got := resource.GetUsagePercent()
			if got != tt.expectedPercent {
				t.Errorf("GetUsagePercent() = %v, want %v", got, tt.expectedPercent)
			}
		})
	}
}

func TestFileStorageResource_CanAddFile(t *testing.T) {
	tests := []struct {
		name       string
		limitBytes int64
		usedBytes  int64
		fileSize   int64
		expected   bool
	}{
		{
			name:       "can add small file",
			limitBytes: 1000,
			usedBytes:  500,
			fileSize:   100,
			expected:   true,
		},
		{
			name:       "can add file up to limit",
			limitBytes: 1000,
			usedBytes:  500,
			fileSize:   500,
			expected:   true,
		},
		{
			name:       "cannot add file exceeding limit",
			limitBytes: 1000,
			usedBytes:  500,
			fileSize:   501,
			expected:   false,
		},
		{
			name:       "cannot add to full storage",
			limitBytes: 1000,
			usedBytes:  1000,
			fileSize:   1,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewFileStorageResource("user-123", "Test Storage")
			resource.StorageLimitBytes = tt.limitBytes
			resource.UsedStorageBytes = tt.usedBytes

			got := resource.CanAddFile(tt.fileSize)
			if got != tt.expected {
				t.Errorf("CanAddFile() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFileStorageResource_AddFile(t *testing.T) {
	tests := []struct {
		name          string
		limitBytes    int64
		usedBytes     int64
		fileSize      int64
		wantErr       bool
		expectedUsed  int64
		expectedCount int
	}{
		{
			name:          "add file successfully",
			limitBytes:    1000,
			usedBytes:     500,
			fileSize:      100,
			wantErr:       false,
			expectedUsed:  600,
			expectedCount: 1,
		},
		{
			name:          "add file up to limit",
			limitBytes:    1000,
			usedBytes:     500,
			fileSize:      500,
			wantErr:       false,
			expectedUsed:  1000,
			expectedCount: 1,
		},
		{
			name:          "cannot add file exceeding limit",
			limitBytes:    1000,
			usedBytes:     500,
			fileSize:      501,
			wantErr:       true,
			expectedUsed:  500,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewFileStorageResource("user-123", "Test Storage")
			resource.StorageLimitBytes = tt.limitBytes
			resource.UsedStorageBytes = tt.usedBytes

			err := resource.AddFile(tt.fileSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if resource.UsedStorageBytes != tt.expectedUsed {
				t.Errorf("UsedStorageBytes = %v, want %v", resource.UsedStorageBytes, tt.expectedUsed)
			}
			if resource.FileCount != tt.expectedCount {
				t.Errorf("FileCount = %v, want %v", resource.FileCount, tt.expectedCount)
			}
		})
	}
}

func TestFileStorageResource_RemoveFile(t *testing.T) {
	resource := NewFileStorageResource("user-123", "Test Storage")
	resource.UsedStorageBytes = 500
	resource.FileCount = 2

	resource.RemoveFile(100)

	if resource.UsedStorageBytes != 400 {
		t.Errorf("UsedStorageBytes = %v, want %v", resource.UsedStorageBytes, 400)
	}
	if resource.FileCount != 1 {
		t.Errorf("FileCount = %v, want %v", resource.FileCount, 1)
	}

	resource.RemoveFile(1000)
	if resource.UsedStorageBytes != 0 {
		t.Errorf("UsedStorageBytes = %v, want %v (should not go negative)", resource.UsedStorageBytes, 0)
	}

	resource.RemoveFile(100)
	if resource.FileCount != 0 {
		t.Errorf("FileCount = %v, want %v (should not go negative)", resource.FileCount, 0)
	}
}

func TestFileStorageResource_GetAvailableSpace(t *testing.T) {
	resource := NewFileStorageResource("user-123", "Test Storage")
	resource.StorageLimitBytes = 1000
	resource.UsedStorageBytes = 300

	available := resource.GetAvailableSpace()
	if available != 700 {
		t.Errorf("GetAvailableSpace() = %v, want %v", available, 700)
	}
}

func TestFileStorageResource_UpdateLimit(t *testing.T) {
	tests := []struct {
		name          string
		currentLimit  int64
		currentUsed   int64
		newLimit      int64
		wantErr       bool
		expectedLimit int64
	}{
		{
			name:          "increase limit",
			currentLimit:  1000,
			currentUsed:   500,
			newLimit:      2000,
			wantErr:       false,
			expectedLimit: 2000,
		},
		{
			name:          "decrease limit above used",
			currentLimit:  1000,
			currentUsed:   500,
			newLimit:      600,
			wantErr:       false,
			expectedLimit: 600,
		},
		{
			name:          "cannot decrease below used",
			currentLimit:  1000,
			currentUsed:   500,
			newLimit:      400,
			wantErr:       true,
			expectedLimit: 1000,
		},
		{
			name:          "cannot set zero limit",
			currentLimit:  1000,
			currentUsed:   0,
			newLimit:      0,
			wantErr:       true,
			expectedLimit: 1000,
		},
		{
			name:          "cannot set negative limit",
			currentLimit:  1000,
			currentUsed:   0,
			newLimit:      -100,
			wantErr:       true,
			expectedLimit: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewFileStorageResource("user-123", "Test Storage")
			resource.StorageLimitBytes = tt.currentLimit
			resource.UsedStorageBytes = tt.currentUsed

			err := resource.UpdateLimit(tt.newLimit)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateLimit() error = %v, wantErr %v", err, tt.wantErr)
			}
			if resource.StorageLimitBytes != tt.expectedLimit {
				t.Errorf("StorageLimitBytes = %v, want %v", resource.StorageLimitBytes, tt.expectedLimit)
			}
		})
	}
}
