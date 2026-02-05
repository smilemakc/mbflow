package serviceapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/pkg/models"
)

// --- ListAuditLog ---

func TestListAuditLog_ShouldReturnLogs_WhenNoFilters(t *testing.T) {
	// Arrange
	auditLogRepo := new(mockAuditLogRepo)
	ops := newTestOperations(nil, nil, nil, nil, auditLogRepo, nil, nil)

	expectedLogs := []*models.ServiceAuditLog{
		{ID: "log-1", ServiceName: "service-a", Action: "create"},
		{ID: "log-2", ServiceName: "service-b", Action: "delete"},
	}
	auditLogRepo.On("FindAll", mock.Anything, mock.AnythingOfType("repository.ServiceAuditLogFilter")).
		Return(expectedLogs, int64(2), nil)

	// Act
	result, err := ops.ListAuditLog(context.Background(), ListAuditLogParams{
		Limit:  50,
		Offset: 0,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.AuditLogs, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, "log-1", result.AuditLogs[0].ID)
	assert.Equal(t, "log-2", result.AuditLogs[1].ID)
	auditLogRepo.AssertExpectations(t)
}

func TestListAuditLog_ShouldCapLimitAt100(t *testing.T) {
	// Arrange
	auditLogRepo := new(mockAuditLogRepo)
	ops := newTestOperations(nil, nil, nil, nil, auditLogRepo, nil, nil)

	auditLogRepo.On("FindAll", mock.Anything, mock.MatchedBy(func(f repository.ServiceAuditLogFilter) bool {
		return f.Limit == 100
	})).Return([]*models.ServiceAuditLog{}, int64(0), nil)

	// Act
	result, err := ops.ListAuditLog(context.Background(), ListAuditLogParams{
		Limit: 500,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	auditLogRepo.AssertExpectations(t)
}

func TestListAuditLog_ShouldPassLimitUnchanged_WhenUnder100(t *testing.T) {
	// Arrange
	auditLogRepo := new(mockAuditLogRepo)
	ops := newTestOperations(nil, nil, nil, nil, auditLogRepo, nil, nil)

	auditLogRepo.On("FindAll", mock.Anything, mock.MatchedBy(func(f repository.ServiceAuditLogFilter) bool {
		return f.Limit == 50
	})).Return([]*models.ServiceAuditLog{}, int64(0), nil)

	// Act
	_, err := ops.ListAuditLog(context.Background(), ListAuditLogParams{
		Limit: 50,
	})

	// Assert
	require.NoError(t, err)
	auditLogRepo.AssertExpectations(t)
}

func TestListAuditLog_ShouldPassAllFilters(t *testing.T) {
	// Arrange
	auditLogRepo := new(mockAuditLogRepo)
	ops := newTestOperations(nil, nil, nil, nil, auditLogRepo, nil, nil)

	svc := "my-svc"
	action := "update"
	resType := "workflow"
	userID := uuid.New()
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	auditLogRepo.On("FindAll", mock.Anything, mock.MatchedBy(func(f repository.ServiceAuditLogFilter) bool {
		return f.Limit == 25 &&
			f.Offset == 10 &&
			f.ServiceName != nil && *f.ServiceName == svc &&
			f.Action != nil && *f.Action == action &&
			f.ResourceType != nil && *f.ResourceType == resType &&
			f.ImpersonatedUserID != nil && *f.ImpersonatedUserID == userID &&
			f.DateFrom != nil && f.DateFrom.Equal(from) &&
			f.DateTo != nil && f.DateTo.Equal(to)
	})).Return([]*models.ServiceAuditLog{}, int64(0), nil)

	// Act
	_, err := ops.ListAuditLog(context.Background(), ListAuditLogParams{
		Limit:              25,
		Offset:             10,
		ServiceName:        &svc,
		Action:             &action,
		ResourceType:       &resType,
		ImpersonatedUserID: &userID,
		DateFrom:           &from,
		DateTo:             &to,
	})

	// Assert
	require.NoError(t, err)
	auditLogRepo.AssertExpectations(t)
}

func TestListAuditLog_ShouldReturnError_WhenServiceFails(t *testing.T) {
	// Arrange
	auditLogRepo := new(mockAuditLogRepo)
	ops := newTestOperations(nil, nil, nil, nil, auditLogRepo, nil, nil)

	auditLogRepo.On("FindAll", mock.Anything, mock.Anything).
		Return(([]*models.ServiceAuditLog)(nil), int64(0), errors.New("db error"))

	// Act
	result, err := ops.ListAuditLog(context.Background(), ListAuditLogParams{
		Limit: 10,
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestListAuditLog_ShouldReturnEmptyResult_WhenNoLogsFound(t *testing.T) {
	// Arrange
	auditLogRepo := new(mockAuditLogRepo)
	ops := newTestOperations(nil, nil, nil, nil, auditLogRepo, nil, nil)

	auditLogRepo.On("FindAll", mock.Anything, mock.Anything).
		Return([]*models.ServiceAuditLog{}, int64(0), nil)

	// Act
	result, err := ops.ListAuditLog(context.Background(), ListAuditLogParams{
		Limit: 10,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.AuditLogs)
	assert.Equal(t, int64(0), result.Total)
}

func TestListAuditLog_ShouldHandleExactly100Limit(t *testing.T) {
	// Arrange
	auditLogRepo := new(mockAuditLogRepo)
	ops := newTestOperations(nil, nil, nil, nil, auditLogRepo, nil, nil)

	auditLogRepo.On("FindAll", mock.Anything, mock.MatchedBy(func(f repository.ServiceAuditLogFilter) bool {
		return f.Limit == 100
	})).Return([]*models.ServiceAuditLog{}, int64(0), nil)

	// Act
	_, err := ops.ListAuditLog(context.Background(), ListAuditLogParams{
		Limit: 100,
	})

	// Assert
	require.NoError(t, err)
	auditLogRepo.AssertExpectations(t)
}

func TestListAuditLog_ShouldHandleZeroLimit(t *testing.T) {
	// Arrange
	auditLogRepo := new(mockAuditLogRepo)
	ops := newTestOperations(nil, nil, nil, nil, auditLogRepo, nil, nil)

	auditLogRepo.On("FindAll", mock.Anything, mock.MatchedBy(func(f repository.ServiceAuditLogFilter) bool {
		return f.Limit == 0
	})).Return([]*models.ServiceAuditLog{}, int64(0), nil)

	// Act
	_, err := ops.ListAuditLog(context.Background(), ListAuditLogParams{
		Limit: 0,
	})

	// Assert
	require.NoError(t, err)
	auditLogRepo.AssertExpectations(t)
}
