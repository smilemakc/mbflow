package models

import (
	"testing"
	"time"
)

func TestBaseResource_Validate(t *testing.T) {
	tests := []struct {
		name     string
		resource *BaseResource
		wantErr  bool
	}{
		{
			name: "valid resource",
			resource: &BaseResource{
				ID:      "res-123",
				Type:    ResourceTypeFileStorage,
				OwnerID: "user-123",
				Name:    "Test Resource",
				Status:  ResourceStatusActive,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			resource: &BaseResource{
				ID:      "res-123",
				Type:    ResourceTypeFileStorage,
				OwnerID: "user-123",
				Status:  ResourceStatusActive,
			},
			wantErr: true,
		},
		{
			name: "missing owner ID",
			resource: &BaseResource{
				ID:     "res-123",
				Type:   ResourceTypeFileStorage,
				Name:   "Test Resource",
				Status: ResourceStatusActive,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resource.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BaseResource.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBaseResource_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   ResourceStatus
		expected bool
	}{
		{
			name:     "active resource",
			status:   ResourceStatusActive,
			expected: true,
		},
		{
			name:     "suspended resource",
			status:   ResourceStatusSuspended,
			expected: false,
		},
		{
			name:     "deleted resource",
			status:   ResourceStatusDeleted,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &BaseResource{
				Status: tt.status,
			}
			if got := resource.IsActive(); got != tt.expected {
				t.Errorf("BaseResource.IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBaseResource_Interface(t *testing.T) {
	now := time.Now()
	resource := &BaseResource{
		ID:        "res-123",
		Type:      ResourceTypeFileStorage,
		OwnerID:   "user-123",
		Name:      "Test Resource",
		Status:    ResourceStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var _ Resource = resource

	if resource.GetID() != "res-123" {
		t.Errorf("GetID() = %v, want %v", resource.GetID(), "res-123")
	}
	if resource.GetType() != ResourceTypeFileStorage {
		t.Errorf("GetType() = %v, want %v", resource.GetType(), ResourceTypeFileStorage)
	}
	if resource.GetOwnerID() != "user-123" {
		t.Errorf("GetOwnerID() = %v, want %v", resource.GetOwnerID(), "user-123")
	}
	if resource.GetName() != "Test Resource" {
		t.Errorf("GetName() = %v, want %v", resource.GetName(), "Test Resource")
	}
	if resource.GetStatus() != ResourceStatusActive {
		t.Errorf("GetStatus() = %v, want %v", resource.GetStatus(), ResourceStatusActive)
	}
}
