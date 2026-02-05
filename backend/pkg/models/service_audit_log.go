package models

import (
	"time"

	"github.com/google/uuid"
)

type ServiceAuditLog struct {
	ID                 string    `json:"id"`
	SystemKeyID        string    `json:"system_key_id"`
	ServiceName        string    `json:"service_name"`
	ImpersonatedUserID *string   `json:"impersonated_user_id,omitempty"`
	Action             string    `json:"action"`
	ResourceType       string    `json:"resource_type"`
	ResourceID         *string   `json:"resource_id,omitempty"`
	RequestMethod      string    `json:"request_method"`
	RequestPath        string    `json:"request_path"`
	RequestBody        *string   `json:"request_body,omitempty"`
	ResponseStatus     int       `json:"response_status"`
	IPAddress          string    `json:"ip_address"`
	CreatedAt          time.Time `json:"created_at"`
}

func NewServiceAuditLog(systemKeyID, serviceName, action, resourceType, requestMethod, requestPath, ipAddress string, responseStatus int) *ServiceAuditLog {
	return &ServiceAuditLog{
		ID:             uuid.New().String(),
		SystemKeyID:    systemKeyID,
		ServiceName:    serviceName,
		Action:         action,
		ResourceType:   resourceType,
		RequestMethod:  requestMethod,
		RequestPath:    requestPath,
		ResponseStatus: responseStatus,
		IPAddress:      ipAddress,
		CreatedAt:      time.Now(),
	}
}
