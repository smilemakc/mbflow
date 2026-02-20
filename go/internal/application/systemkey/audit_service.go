package systemkey

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

const DefaultRetentionDays = 90

var sensitiveFields = []string{"key", "secret", "password", "token", "credential", "hash"}

type AuditService struct {
	repo          repository.ServiceAuditLogRepository
	retentionDays int
}

func NewAuditService(repo repository.ServiceAuditLogRepository, retentionDays int) *AuditService {
	if retentionDays <= 0 {
		retentionDays = DefaultRetentionDays
	}
	return &AuditService{
		repo:          repo,
		retentionDays: retentionDays,
	}
}

func (s *AuditService) LogAction(ctx context.Context, systemKeyID, serviceName, action, resourceType string, resourceID, impersonatedUserID *string, method, path string, body *string, ip string, status int) error {
	entry := models.NewServiceAuditLog(systemKeyID, serviceName, action, resourceType, method, path, ip, status)
	entry.ResourceID = resourceID
	entry.ImpersonatedUserID = impersonatedUserID

	if body != nil {
		sanitized := sanitizeBody(*body)
		entry.RequestBody = &sanitized
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return fmt.Errorf("failed to create audit log entry: %w", err)
	}
	return nil
}

func (s *AuditService) ListLogs(ctx context.Context, filter repository.ServiceAuditLogFilter) ([]*models.ServiceAuditLog, int64, error) {
	logs, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	return logs, total, nil
}

func (s *AuditService) Cleanup(ctx context.Context) (int64, error) {
	before := time.Now().AddDate(0, 0, -s.retentionDays)
	deleted, err := s.repo.DeleteOlderThan(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup audit logs: %w", err)
	}
	return deleted, nil
}

func sanitizeBody(body string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return body
	}

	sanitizeMap(data)

	result, err := json.Marshal(data)
	if err != nil {
		return body
	}
	return string(result)
}

func sanitizeMap(data map[string]any) {
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		isSensitive := false
		for _, sf := range sensitiveFields {
			if strings.Contains(lowerKey, sf) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			data[key] = "[REDACTED]"
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			sanitizeMap(nested)
		}
	}
}
