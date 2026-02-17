package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// ServiceAuditLogModel represents service_audit_log table
type ServiceAuditLogModel struct {
	bun.BaseModel `bun:"table:mbflow_service_audit_log,alias:sal"`

	ID                 uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	SystemKeyID        uuid.UUID  `bun:"system_key_id,notnull,type:uuid"`
	ServiceName        string     `bun:"service_name,notnull"`
	ImpersonatedUserID *uuid.UUID `bun:"impersonated_user_id,type:uuid"`
	Action             string     `bun:"action,notnull"`
	ResourceType       string     `bun:"resource_type,notnull"`
	ResourceID         *uuid.UUID `bun:"resource_id,type:uuid"`
	RequestMethod      string     `bun:"request_method,notnull"`
	RequestPath        string     `bun:"request_path,notnull"`
	RequestBody        *string    `bun:"request_body,type:text"`
	ResponseStatus     int        `bun:"response_status,notnull"`
	IPAddress          string     `bun:"ip_address,notnull"`
	CreatedAt          time.Time  `bun:"created_at,notnull,default:current_timestamp"`
}

// TableName returns the table name for ServiceAuditLogModel
func (ServiceAuditLogModel) TableName() string {
	return "mbflow_service_audit_log"
}

// BeforeInsert hook to set timestamps and defaults
func (s *ServiceAuditLogModel) BeforeInsert(ctx any) error {
	s.CreatedAt = time.Now()
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// ToServiceAuditLogDomain converts DB model to domain model
func (s *ServiceAuditLogModel) ToServiceAuditLogDomain() *pkgmodels.ServiceAuditLog {
	if s == nil {
		return nil
	}

	var impersonatedUserID *string
	if s.ImpersonatedUserID != nil {
		id := s.ImpersonatedUserID.String()
		impersonatedUserID = &id
	}

	var resourceID *string
	if s.ResourceID != nil {
		id := s.ResourceID.String()
		resourceID = &id
	}

	return &pkgmodels.ServiceAuditLog{
		ID:                 s.ID.String(),
		SystemKeyID:        s.SystemKeyID.String(),
		ServiceName:        s.ServiceName,
		ImpersonatedUserID: impersonatedUserID,
		Action:             s.Action,
		ResourceType:       s.ResourceType,
		ResourceID:         resourceID,
		RequestMethod:      s.RequestMethod,
		RequestPath:        s.RequestPath,
		RequestBody:        s.RequestBody,
		ResponseStatus:     s.ResponseStatus,
		IPAddress:          s.IPAddress,
		CreatedAt:          s.CreatedAt,
	}
}

// FromServiceAuditLogDomain creates DB model from domain model
func FromServiceAuditLogDomain(log *pkgmodels.ServiceAuditLog) *ServiceAuditLogModel {
	if log == nil {
		return nil
	}

	var id uuid.UUID
	if log.ID != "" {
		id = uuid.MustParse(log.ID)
	}

	var impersonatedUserID *uuid.UUID
	if log.ImpersonatedUserID != nil {
		parsed := uuid.MustParse(*log.ImpersonatedUserID)
		impersonatedUserID = &parsed
	}

	var resourceID *uuid.UUID
	if log.ResourceID != nil {
		parsed := uuid.MustParse(*log.ResourceID)
		resourceID = &parsed
	}

	return &ServiceAuditLogModel{
		ID:                 id,
		SystemKeyID:        uuid.MustParse(log.SystemKeyID),
		ServiceName:        log.ServiceName,
		ImpersonatedUserID: impersonatedUserID,
		Action:             log.Action,
		ResourceType:       log.ResourceType,
		ResourceID:         resourceID,
		RequestMethod:      log.RequestMethod,
		RequestPath:        log.RequestPath,
		RequestBody:        log.RequestBody,
		ResponseStatus:     log.ResponseStatus,
		IPAddress:          log.IPAddress,
		CreatedAt:          log.CreatedAt,
	}
}
