package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ---------------------------------------------------------------------------
// Mock: SystemKeyRepository
// ---------------------------------------------------------------------------

type mockSystemKeyRepository struct {
	findByPrefixFn  func(ctx context.Context, prefix string) ([]*models.SystemKey, error)
	updateLastUsedFn func(ctx context.Context, id uuid.UUID) error
	countFn         func(ctx context.Context) (int64, error)
}

func (m *mockSystemKeyRepository) Create(_ context.Context, _ *models.SystemKey) error {
	return nil
}

func (m *mockSystemKeyRepository) FindByID(_ context.Context, _ uuid.UUID) (*models.SystemKey, error) {
	return nil, nil
}

func (m *mockSystemKeyRepository) FindByPrefix(ctx context.Context, prefix string) ([]*models.SystemKey, error) {
	if m.findByPrefixFn != nil {
		return m.findByPrefixFn(ctx, prefix)
	}
	return nil, nil
}

func (m *mockSystemKeyRepository) FindAll(_ context.Context, _ repository.SystemKeyFilter) ([]*models.SystemKey, int64, error) {
	return nil, 0, nil
}

func (m *mockSystemKeyRepository) Update(_ context.Context, _ *models.SystemKey) error {
	return nil
}

func (m *mockSystemKeyRepository) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockSystemKeyRepository) Revoke(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockSystemKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	if m.updateLastUsedFn != nil {
		return m.updateLastUsedFn(ctx, id)
	}
	return nil
}

func (m *mockSystemKeyRepository) Count(ctx context.Context) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx)
	}
	return 0, nil
}

// Compile-time check that mockSystemKeyRepository implements the interface.
var _ repository.SystemKeyRepository = (*mockSystemKeyRepository)(nil)

// ---------------------------------------------------------------------------
// Mock: ServiceAuditLogRepository
// ---------------------------------------------------------------------------

type mockServiceAuditLogRepository struct {
	createFn func(ctx context.Context, log *models.ServiceAuditLog) error
}

func (m *mockServiceAuditLogRepository) Create(ctx context.Context, log *models.ServiceAuditLog) error {
	if m.createFn != nil {
		return m.createFn(ctx, log)
	}
	return nil
}

func (m *mockServiceAuditLogRepository) FindAll(_ context.Context, _ repository.ServiceAuditLogFilter) ([]*models.ServiceAuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockServiceAuditLogRepository) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

var _ repository.ServiceAuditLogRepository = (*mockServiceAuditLogRepository)(nil)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestLogger creates a logger suitable for tests (suppresses output).
func newTestLogger() *logger.Logger {
	return logger.New(config.LoggingConfig{Level: "error", Format: "json"})
}

// noopHandler is a gRPC handler that returns "ok" and captures the context.
func noopHandler(capturedCtx *context.Context) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		if capturedCtx != nil {
			*capturedCtx = ctx
		}
		return "ok", nil
	}
}

// failingHandler is a gRPC handler that returns a gRPC error.
func failingHandler(code codes.Code, msg string) grpc.UnaryHandler {
	return func(_ context.Context, _ any) (any, error) {
		return nil, status.Errorf(code, "%s", msg)
	}
}

// testServerInfo builds a minimal *grpc.UnaryServerInfo.
func testServerInfo(fullMethod string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: fullMethod}
}

// newSystemKeyService creates a *systemkey.Service backed by the given mock repository.
func newSystemKeyService(repo repository.SystemKeyRepository) *systemkey.Service {
	return systemkey.NewService(repo, systemkey.Config{
		MaxKeys:    100,
		BcryptCost: 4, // minimal cost for fast tests
	})
}

// ctxWithMetadata creates a context with the given gRPC incoming metadata key-value pairs.
func ctxWithMetadata(kvs ...string) context.Context {
	m := make(map[string]string)
	for i := 0; i+1 < len(kvs); i += 2 {
		m[kvs[i]] = kvs[i+1]
	}
	md := metadata.New(m)
	return metadata.NewIncomingContext(context.Background(), md)
}

// newBunDBWithMock creates a bun.DB backed by go-sqlmock for unit testing.
// It registers the bun models needed for the UserRepository queries.
// Uses QueryMatcherRegexp so that ExpectQuery patterns are treated as regexps.
func newBunDBWithMock(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	bunDB := bun.NewDB(db, pgdialect.New())
	// Register the m2m model so bun can resolve UserModel relations without panicking.
	bunDB.RegisterModel((*storagemodels.UserRoleModel)(nil))
	return bunDB, mock
}

// ==========================================================================
// Tests: extractSystemKeyFromMetadata
// ==========================================================================

func TestExtractSystemKeyFromMetadata_ShouldReturnKey_WhenXSystemKeyHeaderPresent(t *testing.T) {
	// Arrange
	ctx := ctxWithMetadata("x-system-key", "sysk_abc123")

	// Act
	token, err := extractSystemKeyFromMetadata(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sysk_abc123", token)
}

func TestExtractSystemKeyFromMetadata_ShouldReturnKey_WhenBearerAuthorizationPresent(t *testing.T) {
	// Arrange
	ctx := ctxWithMetadata("authorization", "Bearer sysk_abc123")

	// Act
	token, err := extractSystemKeyFromMetadata(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sysk_abc123", token)
}

func TestExtractSystemKeyFromMetadata_ShouldReturnKey_WhenBearerAuthorizationCaseInsensitive(t *testing.T) {
	// Arrange
	ctx := ctxWithMetadata("authorization", "bearer sysk_abc123")

	// Act
	token, err := extractSystemKeyFromMetadata(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sysk_abc123", token)
}

func TestExtractSystemKeyFromMetadata_ShouldReturnError_WhenNoMetadata(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// Act
	_, err := extractSystemKeyFromMetadata(ctx)

	// Assert
	require.Error(t, err)
}

func TestExtractSystemKeyFromMetadata_ShouldReturnError_WhenEmptyMetadata(t *testing.T) {
	// Arrange
	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Act
	_, err := extractSystemKeyFromMetadata(ctx)

	// Assert
	require.Error(t, err)
}

func TestExtractSystemKeyFromMetadata_ShouldReturnError_WhenAuthorizationNotBearer(t *testing.T) {
	// Arrange
	ctx := ctxWithMetadata("authorization", "Basic sysk_abc123")

	// Act
	_, err := extractSystemKeyFromMetadata(ctx)

	// Assert
	require.Error(t, err)
}

func TestExtractSystemKeyFromMetadata_ShouldReturnError_WhenBearerTokenNotSystemKey(t *testing.T) {
	// Arrange
	ctx := ctxWithMetadata("authorization", "Bearer some_other_token")

	// Act
	_, err := extractSystemKeyFromMetadata(ctx)

	// Assert
	require.Error(t, err)
}

func TestExtractSystemKeyFromMetadata_ShouldPreferXSystemKeyHeader(t *testing.T) {
	// Arrange: both headers present
	ctx := ctxWithMetadata(
		"x-system-key", "sysk_from_header",
		"authorization", "Bearer sysk_from_auth",
	)

	// Act
	token, err := extractSystemKeyFromMetadata(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sysk_from_header", token)
}

// ==========================================================================
// Tests: getMetadataValue
// ==========================================================================

func TestGetMetadataValue_ShouldReturnValue_WhenKeyExists(t *testing.T) {
	ctx := ctxWithMetadata("x-custom-key", "custom-value")
	assert.Equal(t, "custom-value", getMetadataValue(ctx, "x-custom-key"))
}

func TestGetMetadataValue_ShouldReturnEmpty_WhenKeyMissing(t *testing.T) {
	ctx := ctxWithMetadata("x-other", "value")
	assert.Equal(t, "", getMetadataValue(ctx, "x-custom-key"))
}

func TestGetMetadataValue_ShouldReturnEmpty_WhenNoMetadata(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "", getMetadataValue(ctx, "x-custom-key"))
}

// ==========================================================================
// Tests: SystemKeyAuthInterceptor
// ==========================================================================

func TestSystemKeyAuthInterceptor_ShouldAuthenticate_WhenValidXSystemKeyHeader(t *testing.T) {
	// Arrange
	keyID := uuid.New().String()
	serviceName := "test-service"

	// Create a real key through the service to get a valid bcrypt hash
	captureRepo := &capturingSystemKeyRepository{}
	createSvc := newSystemKeyService(captureRepo)

	result, err := createSvc.CreateKey(context.Background(), "test", "desc", serviceName, uuid.New(), nil)
	require.NoError(t, err)
	result.Key.ID = keyID

	// Set up the mock repo for the interceptor to find this key
	repo := &mockSystemKeyRepository{
		findByPrefixFn: func(_ context.Context, _ string) ([]*models.SystemKey, error) {
			return []*models.SystemKey{result.Key}, nil
		},
		updateLastUsedFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	interceptorSvc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(interceptorSvc)

	var capturedCtx context.Context
	handler := noopHandler(&capturedCtx)
	ctx := ctxWithMetadata("x-system-key", result.PlainKey)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	resp, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	// Verify context values
	gotKeyID, ok := SystemKeyIDFromContext(capturedCtx)
	assert.True(t, ok)
	assert.Equal(t, keyID, gotKeyID)

	gotServiceName, ok := ServiceNameFromContext(capturedCtx)
	assert.True(t, ok)
	assert.Equal(t, serviceName, gotServiceName)

	gotIsAdmin, ok := capturedCtx.Value(ctxKeyIsAdmin).(bool)
	assert.True(t, ok)
	assert.True(t, gotIsAdmin)

	gotAuthMethod, ok := capturedCtx.Value(ctxKeyAuthMethod).(string)
	assert.True(t, ok)
	assert.Equal(t, "system_key", gotAuthMethod)
}

func TestSystemKeyAuthInterceptor_ShouldAuthenticate_WhenValidBearerAuthorization(t *testing.T) {
	// Arrange
	captureRepo := &capturingSystemKeyRepository{}
	createSvc := newSystemKeyService(captureRepo)

	result, err := createSvc.CreateKey(context.Background(), "test", "desc", "my-service", uuid.New(), nil)
	require.NoError(t, err)

	repo := &mockSystemKeyRepository{
		findByPrefixFn: func(_ context.Context, _ string) ([]*models.SystemKey, error) {
			return []*models.SystemKey{result.Key}, nil
		},
		updateLastUsedFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	interceptorSvc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(interceptorSvc)

	var capturedCtx context.Context
	handler := noopHandler(&capturedCtx)
	ctx := ctxWithMetadata("authorization", "Bearer "+result.PlainKey)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/GetWorkflow")

	// Act
	resp, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	gotServiceName, ok := ServiceNameFromContext(capturedCtx)
	assert.True(t, ok)
	assert.Equal(t, "my-service", gotServiceName)
}

func TestSystemKeyAuthInterceptor_ShouldReturnUnauthenticated_WhenNoSystemKey(t *testing.T) {
	// Arrange
	repo := &mockSystemKeyRepository{}
	svc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(svc)

	handler := noopHandler(nil)
	ctx := context.Background() // no metadata at all
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "system key required")
}

func TestSystemKeyAuthInterceptor_ShouldReturnUnauthenticated_WhenEmptyMetadata(t *testing.T) {
	// Arrange
	repo := &mockSystemKeyRepository{}
	svc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(svc)

	handler := noopHandler(nil)
	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestSystemKeyAuthInterceptor_ShouldReturnUnauthenticated_WhenInvalidKeyFormat(t *testing.T) {
	// Arrange: key does not start with "sysk_"
	repo := &mockSystemKeyRepository{}
	svc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(svc)

	handler := noopHandler(nil)
	ctx := ctxWithMetadata("x-system-key", "invalid_key_format_1234567890")
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "system key required")
}

func TestSystemKeyAuthInterceptor_ShouldReturnUnauthenticated_WhenKeyRevoked(t *testing.T) {
	// Arrange: create a key, then revoke it
	captureRepo := &capturingSystemKeyRepository{}
	createSvc := newSystemKeyService(captureRepo)

	result, err := createSvc.CreateKey(context.Background(), "test", "desc", "service", uuid.New(), nil)
	require.NoError(t, err)

	// Mark the key as revoked
	result.Key.Revoke()

	repo := &mockSystemKeyRepository{
		findByPrefixFn: func(_ context.Context, _ string) ([]*models.SystemKey, error) {
			return []*models.SystemKey{result.Key}, nil
		},
		updateLastUsedFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	interceptorSvc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(interceptorSvc)

	handler := noopHandler(nil)
	ctx := ctxWithMetadata("x-system-key", result.PlainKey)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err = interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "revoked")
}

func TestSystemKeyAuthInterceptor_ShouldReturnUnauthenticated_WhenKeyExpired(t *testing.T) {
	// Arrange: create a key, then set its expiration in the past
	captureRepo := &capturingSystemKeyRepository{}
	createSvc := newSystemKeyService(captureRepo)

	result, err := createSvc.CreateKey(context.Background(), "test", "desc", "service", uuid.New(), nil)
	require.NoError(t, err)

	// Set expiration in the past
	pastTime := time.Now().Add(-24 * time.Hour)
	result.Key.ExpiresAt = &pastTime

	repo := &mockSystemKeyRepository{
		findByPrefixFn: func(_ context.Context, _ string) ([]*models.SystemKey, error) {
			return []*models.SystemKey{result.Key}, nil
		},
		updateLastUsedFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	interceptorSvc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(interceptorSvc)

	handler := noopHandler(nil)
	ctx := ctxWithMetadata("x-system-key", result.PlainKey)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err = interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "expired")
}

func TestSystemKeyAuthInterceptor_ShouldReturnUnauthenticated_WhenKeyNotFound(t *testing.T) {
	// Arrange: repo returns no keys for the prefix
	repo := &mockSystemKeyRepository{
		findByPrefixFn: func(_ context.Context, _ string) ([]*models.SystemKey, error) {
			return nil, nil
		},
	}

	svc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(svc)

	handler := noopHandler(nil)
	ctx := ctxWithMetadata("x-system-key", "sysk_nonexistent1234567890abcde")
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestSystemKeyAuthInterceptor_ShouldReturnUnauthenticated_WhenRepoFails(t *testing.T) {
	// Arrange: repo returns an error
	repo := &mockSystemKeyRepository{
		findByPrefixFn: func(_ context.Context, _ string) ([]*models.SystemKey, error) {
			return nil, errors.New("database connection lost")
		},
	}

	svc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(svc)

	handler := noopHandler(nil)
	ctx := ctxWithMetadata("x-system-key", "sysk_some_key_1234567890abcdef")
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestSystemKeyAuthInterceptor_ShouldSetAllContextValues(t *testing.T) {
	// Arrange
	captureRepo := &capturingSystemKeyRepository{}
	createSvc := newSystemKeyService(captureRepo)

	result, err := createSvc.CreateKey(context.Background(), "ctx-test", "desc", "ctx-service", uuid.New(), nil)
	require.NoError(t, err)

	repo := &mockSystemKeyRepository{
		findByPrefixFn: func(_ context.Context, _ string) ([]*models.SystemKey, error) {
			return []*models.SystemKey{result.Key}, nil
		},
		updateLastUsedFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	svc := newSystemKeyService(repo)
	interceptor := SystemKeyAuthInterceptor(svc)

	var capturedCtx context.Context
	handler := noopHandler(&capturedCtx)
	ctx := ctxWithMetadata("x-system-key", result.PlainKey)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err = interceptor(ctx, "request", info, handler)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, capturedCtx)

	// auth_method
	authMethod, ok := capturedCtx.Value(ctxKeyAuthMethod).(string)
	assert.True(t, ok, "auth_method should be set")
	assert.Equal(t, "system_key", authMethod)

	// system_key_id
	sysKeyID, ok := SystemKeyIDFromContext(capturedCtx)
	assert.True(t, ok, "system_key_id should be set")
	assert.Equal(t, result.Key.ID, sysKeyID)

	// service_name
	svcName, ok := ServiceNameFromContext(capturedCtx)
	assert.True(t, ok, "service_name should be set")
	assert.Equal(t, "ctx-service", svcName)

	// is_admin
	isAdmin, ok := capturedCtx.Value(ctxKeyIsAdmin).(bool)
	assert.True(t, ok, "is_admin should be set")
	assert.True(t, isAdmin)
}

// ==========================================================================
// Tests: ImpersonationInterceptor
// ==========================================================================

func TestImpersonationInterceptor_ShouldUseSystemUserID_WhenNoOnBehalfOfHeader(t *testing.T) {
	// Arrange
	systemUserID := uuid.New().String()
	bunDB, _ := newBunDBWithMock(t)
	userRepo := storage.NewUserRepository(bunDB)

	interceptor := ImpersonationInterceptor(userRepo, systemUserID)

	var capturedCtx context.Context
	handler := noopHandler(&capturedCtx)
	// Metadata without x-on-behalf-of
	ctx := ctxWithMetadata("x-system-key", "sysk_whatever")
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	resp, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	userID, ok := UserIDFromContext(capturedCtx)
	assert.True(t, ok)
	assert.Equal(t, systemUserID, userID)

	assert.False(t, ImpersonatedFromContext(capturedCtx))
}

func TestImpersonationInterceptor_ShouldSetImpersonatedUser_WhenValidOnBehalfOfHeader(t *testing.T) {
	// Arrange
	systemUserID := uuid.New().String()
	targetUserID := uuid.New()
	now := time.Now()

	bunDB, mock := newBunDBWithMock(t)
	userRepo := storage.NewUserRepository(bunDB)

	// Bun's pgdialect generates SELECT with all struct fields. sqlmock must
	// return rows whose column names match what bun expects after SQL parsing.
	// Bun uses the struct bun-tag names as DB column names during scanning.
	columns := []string{
		"id", "email", "username", "password_hash", "full_name",
		"is_active", "is_admin", "email_verified", "failed_login_attempts",
		"locked_until", "metadata", "created_at", "updated_at", "last_login_at", "deleted_at",
	}
	rows := sqlmock.NewRows(columns).
		AddRow(
			targetUserID, "user@example.com", "testuser", "hash", "Test User",
			true, false, true, 0,
			nil, []byte("{}"), now, now, nil, nil,
		)

	mock.ExpectQuery("^SELECT").WillReturnRows(rows)

	interceptor := ImpersonationInterceptor(userRepo, systemUserID)

	var capturedCtx context.Context
	handler := noopHandler(&capturedCtx)
	ctx := ctxWithMetadata("x-on-behalf-of", targetUserID.String())
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	resp, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	userID, ok := UserIDFromContext(capturedCtx)
	assert.True(t, ok)
	assert.Equal(t, targetUserID.String(), userID)

	assert.True(t, ImpersonatedFromContext(capturedCtx))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImpersonationInterceptor_ShouldReturnInvalidArgument_WhenOnBehalfOfNotUUID(t *testing.T) {
	// Arrange
	systemUserID := uuid.New().String()
	bunDB, _ := newBunDBWithMock(t)
	userRepo := storage.NewUserRepository(bunDB)

	interceptor := ImpersonationInterceptor(userRepo, systemUserID)

	handler := noopHandler(nil)
	ctx := ctxWithMetadata("x-on-behalf-of", "not-a-valid-uuid")
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid user ID format")
}

func TestImpersonationInterceptor_ShouldReturnInvalidArgument_WhenUserNotFound(t *testing.T) {
	// Arrange
	systemUserID := uuid.New().String()
	nonExistentUserID := uuid.New()

	bunDB, mock := newBunDBWithMock(t)
	userRepo := storage.NewUserRepository(bunDB)

	// Return empty result set (user not found).
	// When bun gets sql.ErrNoRows, the repository returns (nil, nil).
	rows := sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "full_name",
		"is_active", "is_admin", "email_verified", "failed_login_attempts",
		"locked_until", "metadata", "created_at", "updated_at", "last_login_at", "deleted_at"})

	mock.ExpectQuery("^SELECT").
		WillReturnRows(rows)

	interceptor := ImpersonationInterceptor(userRepo, systemUserID)

	handler := noopHandler(nil)
	ctx := ctxWithMetadata("x-on-behalf-of", nonExistentUserID.String())
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "user not found")
}

func TestImpersonationInterceptor_ShouldReturnInternal_WhenDBFails(t *testing.T) {
	// Arrange
	systemUserID := uuid.New().String()
	targetUserID := uuid.New()

	bunDB, mock := newBunDBWithMock(t)
	userRepo := storage.NewUserRepository(bunDB)

	mock.ExpectQuery("^SELECT").
		WillReturnError(fmt.Errorf("connection refused"))

	interceptor := ImpersonationInterceptor(userRepo, systemUserID)

	handler := noopHandler(nil)
	ctx := ctxWithMetadata("x-on-behalf-of", targetUserID.String())
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "failed to validate user")
}

func TestImpersonationInterceptor_ShouldNotSetImpersonated_WhenNoHeader(t *testing.T) {
	// Arrange
	systemUserID := uuid.New().String()
	bunDB, _ := newBunDBWithMock(t)
	userRepo := storage.NewUserRepository(bunDB)

	interceptor := ImpersonationInterceptor(userRepo, systemUserID)

	var capturedCtx context.Context
	handler := noopHandler(&capturedCtx)
	// Empty metadata (no x-on-behalf-of)
	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.NoError(t, err)

	impersonated := ImpersonatedFromContext(capturedCtx)
	assert.False(t, impersonated)

	userID, ok := UserIDFromContext(capturedCtx)
	assert.True(t, ok)
	assert.Equal(t, systemUserID, userID)
}

// ==========================================================================
// Tests: AuditInterceptor
// ==========================================================================

func TestAuditInterceptor_ShouldCallHandler_AndLogAudit(t *testing.T) {
	// Arrange
	auditLogCh := make(chan *models.ServiceAuditLog, 1)
	auditRepo := &mockServiceAuditLogRepository{
		createFn: func(_ context.Context, log *models.ServiceAuditLog) error {
			auditLogCh <- log
			return nil
		},
	}
	auditService := systemkey.NewAuditService(auditRepo, 90)
	log := newTestLogger()

	interceptor := AuditInterceptor(auditService, log)

	// Build a context with auth info already set (as would be done by prior interceptors)
	ctx := ctxWithMetadata("x-system-key", "sysk_test")
	ctx = ContextWithSystemKeyID(ctx, "key-123")
	ctx = ContextWithServiceName(ctx, "test-service")
	ctx = ContextWithUserID(ctx, uuid.New().String())
	ctx = ContextWithImpersonated(ctx, false)

	handler := noopHandler(nil)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/ListWorkflows")

	// Act
	resp, err := interceptor(ctx, "request", info, handler)

	// Assert: handler should succeed
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	// Wait for the async audit log
	select {
	case entry := <-auditLogCh:
		assert.Equal(t, "key-123", entry.SystemKeyID)
		assert.Equal(t, "test-service", entry.ServiceName)
		assert.Equal(t, "workflow.list", entry.Action)
		assert.Equal(t, "workflow", entry.ResourceType)
		assert.Equal(t, "gRPC", entry.RequestMethod)
		assert.Equal(t, 200, entry.ResponseStatus)
		assert.Nil(t, entry.ImpersonatedUserID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log entry")
	}
}

func TestAuditInterceptor_ShouldLogErrorStatus_WhenHandlerFails(t *testing.T) {
	// Arrange
	auditLogCh := make(chan *models.ServiceAuditLog, 1)
	auditRepo := &mockServiceAuditLogRepository{
		createFn: func(_ context.Context, log *models.ServiceAuditLog) error {
			auditLogCh <- log
			return nil
		},
	}
	auditService := systemkey.NewAuditService(auditRepo, 90)
	log := newTestLogger()

	interceptor := AuditInterceptor(auditService, log)

	ctx := ctxWithMetadata("x-system-key", "sysk_test")
	ctx = ContextWithSystemKeyID(ctx, "key-456")
	ctx = ContextWithServiceName(ctx, "failing-service")

	handler := failingHandler(codes.NotFound, "workflow not found")
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/GetWorkflow")

	// Act
	resp, err := interceptor(ctx, "request", info, handler)

	// Assert: should propagate the handler error
	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())

	// Wait for the async audit log
	select {
	case entry := <-auditLogCh:
		assert.Equal(t, "key-456", entry.SystemKeyID)
		assert.Equal(t, "workflow.get", entry.Action)
		assert.Equal(t, 404, entry.ResponseStatus) // NotFound -> 404
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log entry")
	}
}

func TestAuditInterceptor_ShouldIncludeImpersonatedUserID_WhenImpersonated(t *testing.T) {
	// Arrange
	auditLogCh := make(chan *models.ServiceAuditLog, 1)
	auditRepo := &mockServiceAuditLogRepository{
		createFn: func(_ context.Context, log *models.ServiceAuditLog) error {
			auditLogCh <- log
			return nil
		},
	}
	auditService := systemkey.NewAuditService(auditRepo, 90)
	log := newTestLogger()

	interceptor := AuditInterceptor(auditService, log)

	impersonatedUserID := uuid.New().String()
	ctx := ctxWithMetadata("x-system-key", "sysk_test")
	ctx = ContextWithSystemKeyID(ctx, "key-789")
	ctx = ContextWithServiceName(ctx, "impersonating-service")
	ctx = ContextWithUserID(ctx, impersonatedUserID)
	ctx = ContextWithImpersonated(ctx, true)

	handler := noopHandler(nil)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/CreateWorkflow")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.NoError(t, err)

	select {
	case entry := <-auditLogCh:
		require.NotNil(t, entry.ImpersonatedUserID)
		assert.Equal(t, impersonatedUserID, *entry.ImpersonatedUserID)
		assert.Equal(t, "workflow.create", entry.Action)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log entry")
	}
}

func TestAuditInterceptor_ShouldNotIncludeImpersonatedUserID_WhenNotImpersonated(t *testing.T) {
	// Arrange
	auditLogCh := make(chan *models.ServiceAuditLog, 1)
	auditRepo := &mockServiceAuditLogRepository{
		createFn: func(_ context.Context, log *models.ServiceAuditLog) error {
			auditLogCh <- log
			return nil
		},
	}
	auditService := systemkey.NewAuditService(auditRepo, 90)
	log := newTestLogger()

	interceptor := AuditInterceptor(auditService, log)

	ctx := ctxWithMetadata("x-system-key", "sysk_test")
	ctx = ContextWithSystemKeyID(ctx, "key-abc")
	ctx = ContextWithServiceName(ctx, "normal-service")
	ctx = ContextWithUserID(ctx, uuid.New().String())
	ctx = ContextWithImpersonated(ctx, false)

	handler := noopHandler(nil)
	info := testServerInfo("/serviceapi.MBFlowServiceAPI/DeleteWorkflow")

	// Act
	_, err := interceptor(ctx, "request", info, handler)

	// Assert
	require.NoError(t, err)

	select {
	case entry := <-auditLogCh:
		assert.Nil(t, entry.ImpersonatedUserID)
		assert.Equal(t, "workflow.delete", entry.Action)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log entry")
	}
}

// ==========================================================================
// Tests: parseGRPCMethod
// ==========================================================================

func TestParseGRPCMethod_ShouldParseKnownMethods(t *testing.T) {
	tests := []struct {
		name             string
		fullMethod       string
		expectedAction   string
		expectedResource string
	}{
		// Workflow operations
		{
			name:             "ListWorkflows",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/ListWorkflows",
			expectedAction:   "workflow.list",
			expectedResource: "workflow",
		},
		{
			name:             "GetWorkflow",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/GetWorkflow",
			expectedAction:   "workflow.get",
			expectedResource: "workflow",
		},
		{
			name:             "CreateWorkflow",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/CreateWorkflow",
			expectedAction:   "workflow.create",
			expectedResource: "workflow",
		},
		{
			name:             "UpdateWorkflow",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/UpdateWorkflow",
			expectedAction:   "workflow.update",
			expectedResource: "workflow",
		},
		{
			name:             "DeleteWorkflow",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/DeleteWorkflow",
			expectedAction:   "workflow.delete",
			expectedResource: "workflow",
		},

		// Execution operations
		{
			name:             "ListExecutions",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/ListExecutions",
			expectedAction:   "execution.list",
			expectedResource: "execution",
		},
		{
			name:             "GetExecution",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/GetExecution",
			expectedAction:   "execution.get",
			expectedResource: "execution",
		},
		{
			name:             "StartExecution",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/StartExecution",
			expectedAction:   "execution.start",
			expectedResource: "execution",
		},
		{
			name:             "CancelExecution",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/CancelExecution",
			expectedAction:   "execution.cancel",
			expectedResource: "execution",
		},
		{
			name:             "RetryExecution",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/RetryExecution",
			expectedAction:   "execution.retry",
			expectedResource: "execution",
		},

		// Trigger operations
		{
			name:             "ListTriggers",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/ListTriggers",
			expectedAction:   "trigger.list",
			expectedResource: "trigger",
		},
		{
			name:             "CreateTrigger",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/CreateTrigger",
			expectedAction:   "trigger.create",
			expectedResource: "trigger",
		},
		{
			name:             "UpdateTrigger",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/UpdateTrigger",
			expectedAction:   "trigger.update",
			expectedResource: "trigger",
		},
		{
			name:             "DeleteTrigger",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/DeleteTrigger",
			expectedAction:   "trigger.delete",
			expectedResource: "trigger",
		},

		// Credential operations
		{
			name:             "ListCredentials",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/ListCredentials",
			expectedAction:   "credential.list",
			expectedResource: "credential",
		},
		{
			name:             "CreateCredential",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/CreateCredential",
			expectedAction:   "credential.create",
			expectedResource: "credential",
		},
		{
			name:             "UpdateCredential",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/UpdateCredential",
			expectedAction:   "credential.update",
			expectedResource: "credential",
		},
		{
			name:             "DeleteCredential",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/DeleteCredential",
			expectedAction:   "credential.delete",
			expectedResource: "credential",
		},

		// Audit log
		{
			name:             "ListAuditLog",
			fullMethod:       "/serviceapi.MBFlowServiceAPI/ListAuditLog",
			expectedAction:   "audit_log.list",
			expectedResource: "audit_log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, resourceType, resourceID := parseGRPCMethod(tt.fullMethod)

			assert.Equal(t, tt.expectedAction, action)
			assert.Equal(t, tt.expectedResource, resourceType)
			assert.Nil(t, resourceID)
		})
	}
}

func TestParseGRPCMethod_ShouldReturnUnknown_WhenMethodNotRecognized(t *testing.T) {
	// Arrange
	fullMethod := "/serviceapi.MBFlowServiceAPI/SomeCustomMethod"

	// Act
	action, resourceType, resourceID := parseGRPCMethod(fullMethod)

	// Assert
	assert.Equal(t, "unknown.somecustommethod", action)
	assert.Equal(t, "unknown", resourceType)
	assert.Nil(t, resourceID)
}

func TestParseGRPCMethod_ShouldReturnUnknown_WhenMethodPathTooShort(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
	}{
		{name: "empty string", fullMethod: ""},
		{name: "single slash", fullMethod: "/"},
		{name: "only service", fullMethod: "/serviceapi.MBFlowServiceAPI"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, resourceType, resourceID := parseGRPCMethod(tt.fullMethod)

			assert.Equal(t, "unknown", action)
			assert.Equal(t, "unknown", resourceType)
			assert.Nil(t, resourceID)
		})
	}
}

func TestParseGRPCMethod_ShouldHandleUnknownPrefix_WhenResourceUnrecognized(t *testing.T) {
	// A method with a known prefix pattern (List) but unknown resource
	fullMethod := "/serviceapi.MBFlowServiceAPI/ListFoobar"

	action, resourceType, resourceID := parseGRPCMethod(fullMethod)

	// Resource type defaults to unknown, but action uses the List prefix
	assert.Equal(t, "unknown.list", action)
	assert.Equal(t, "unknown", resourceType)
	assert.Nil(t, resourceID)
}

func TestParseGRPCMethod_ShouldHandleDifferentServiceNames(t *testing.T) {
	fullMethod := "/otherpackage.OtherService/ListWorkflows"

	action, resourceType, _ := parseGRPCMethod(fullMethod)

	assert.Equal(t, "workflow.list", action)
	assert.Equal(t, "workflow", resourceType)
}

// ==========================================================================
// Tests: grpcCodeToHTTPStatus
// ==========================================================================

func TestGrpcCodeToHTTPStatus_ShouldMapAllDefinedCodes(t *testing.T) {
	tests := []struct {
		name           string
		code           codes.Code
		expectedStatus int
	}{
		{name: "OK", code: codes.OK, expectedStatus: 200},
		{name: "InvalidArgument", code: codes.InvalidArgument, expectedStatus: 400},
		{name: "Unauthenticated", code: codes.Unauthenticated, expectedStatus: 401},
		{name: "PermissionDenied", code: codes.PermissionDenied, expectedStatus: 403},
		{name: "NotFound", code: codes.NotFound, expectedStatus: 404},
		{name: "AlreadyExists", code: codes.AlreadyExists, expectedStatus: 409},
		{name: "Unimplemented", code: codes.Unimplemented, expectedStatus: 501},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grpcCodeToHTTPStatus(tt.code)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

func TestGrpcCodeToHTTPStatus_ShouldReturn500_ForUnmappedCodes(t *testing.T) {
	unmappedCodes := []codes.Code{
		codes.Canceled,
		codes.Unknown,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Internal,
		codes.Unavailable,
		codes.DataLoss,
	}

	for _, code := range unmappedCodes {
		t.Run(code.String(), func(t *testing.T) {
			result := grpcCodeToHTTPStatus(code)
			assert.Equal(t, 500, result, "unmapped code %s should return 500", code.String())
		})
	}
}

// ---------------------------------------------------------------------------
// capturingSystemKeyRepository: a mock that actually stores created keys
// for use in test key generation. It delegates all operations to simple maps.
// ---------------------------------------------------------------------------

type capturingSystemKeyRepository struct {
	keys []*models.SystemKey
}

func (r *capturingSystemKeyRepository) Create(_ context.Context, key *models.SystemKey) error {
	r.keys = append(r.keys, key)
	return nil
}

func (r *capturingSystemKeyRepository) FindByID(_ context.Context, id uuid.UUID) (*models.SystemKey, error) {
	for _, k := range r.keys {
		if k.ID == id.String() {
			return k, nil
		}
	}
	return nil, nil
}

func (r *capturingSystemKeyRepository) FindByPrefix(_ context.Context, prefix string) ([]*models.SystemKey, error) {
	var result []*models.SystemKey
	for _, k := range r.keys {
		if k.KeyPrefix == prefix {
			result = append(result, k)
		}
	}
	return result, nil
}

func (r *capturingSystemKeyRepository) FindAll(_ context.Context, _ repository.SystemKeyFilter) ([]*models.SystemKey, int64, error) {
	return r.keys, int64(len(r.keys)), nil
}

func (r *capturingSystemKeyRepository) Update(_ context.Context, _ *models.SystemKey) error {
	return nil
}

func (r *capturingSystemKeyRepository) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *capturingSystemKeyRepository) Revoke(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *capturingSystemKeyRepository) UpdateLastUsed(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *capturingSystemKeyRepository) Count(_ context.Context) (int64, error) {
	return int64(len(r.keys)), nil
}

var _ repository.SystemKeyRepository = (*capturingSystemKeyRepository)(nil)
