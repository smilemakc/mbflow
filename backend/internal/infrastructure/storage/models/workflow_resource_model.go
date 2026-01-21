package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// WorkflowResourceModel represents the workflow_resources table
type WorkflowResourceModel struct {
	bun.BaseModel `bun:"table:mbflow_workflow_resources,alias:wr"`

	WorkflowID uuid.UUID  `bun:"workflow_id,pk,type:uuid" json:"workflow_id"`
	ResourceID uuid.UUID  `bun:"resource_id,pk,type:uuid" json:"resource_id"`
	Alias      string     `bun:"alias,notnull" json:"alias" validate:"required,max=100"`
	AccessType string     `bun:"access_type,notnull,default:'read'" json:"access_type" validate:"required,oneof=read write admin"`
	AssignedAt time.Time  `bun:"assigned_at,notnull,default:current_timestamp" json:"assigned_at"`
	AssignedBy *uuid.UUID `bun:"assigned_by,type:uuid" json:"assigned_by,omitempty"`

	// Relations
	Workflow *WorkflowModel `bun:"rel:belongs-to,join:workflow_id=id" json:"workflow,omitempty"`
	Resource *ResourceModel `bun:"rel:belongs-to,join:resource_id=id" json:"resource,omitempty"`
}

// TableName returns the table name for WorkflowResourceModel
func (WorkflowResourceModel) TableName() string {
	return "mbflow_workflow_resources"
}

// BeforeInsert hook to set timestamps and defaults
func (wr *WorkflowResourceModel) BeforeInsert(ctx interface{}) error {
	wr.AssignedAt = time.Now()
	if wr.AccessType == "" {
		wr.AccessType = "read"
	}
	return nil
}

// IsReadOnly returns true if access type is read
func (wr *WorkflowResourceModel) IsReadOnly() bool {
	return wr.AccessType == "read"
}

// IsWritable returns true if access type is write or admin
func (wr *WorkflowResourceModel) IsWritable() bool {
	return wr.AccessType == "write" || wr.AccessType == "admin"
}

// IsAdmin returns true if access type is admin
func (wr *WorkflowResourceModel) IsAdmin() bool {
	return wr.AccessType == "admin"
}
