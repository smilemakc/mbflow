package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func setupFileRepoTest(t *testing.T) (*FileRepository, *bun.DB, func()) {
	ctx := context.Background()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "mbflow_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := postgres.Host(ctx)
	require.NoError(t, err)

	port, err := postgres.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Connect to database
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/mbflow_test?sslmode=disable", host, port.Port())

	// Wait a bit for the database to be fully ready
	time.Sleep(500 * time.Millisecond)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())

	// Run migrations
	migrator, err := NewMigrator(db, migrations.FS)
	require.NoError(t, err)
	err = migrator.Init(ctx)
	require.NoError(t, err)
	err = migrator.Up(ctx)
	require.NoError(t, err)

	repo := NewFileRepository(db)

	cleanup := func() {
		db.Close()
		postgres.Terminate(ctx)
	}

	return repo, db, cleanup
}

// ========== CREATE TESTS ==========

func TestFileRepo_Create_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	file := &models.FileModel{
		StorageID:   "local",
		Name:        "test.txt",
		Path:        "/uploads/test.txt",
		MimeType:    "text/plain",
		Size:        1024,
		Checksum:    "abc123",
		AccessScope: "workflow",
	}

	err := repo.Create(context.Background(), file)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, file.ID)
}

func TestFileRepo_Create_WithTTL(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	ttl := 3600 // 1 hour
	expiresAt := time.Now().Add(time.Hour)

	file := &models.FileModel{
		StorageID:  "local",
		Name:       "temp.txt",
		Path:       "/temp/temp.txt",
		MimeType:   "text/plain",
		Size:       512,
		Checksum:   "xyz789",
		TTLSeconds: &ttl,
		ExpiresAt:  &expiresAt,
	}

	err := repo.Create(context.Background(), file)
	require.NoError(t, err)
	assert.NotNil(t, file.TTLSeconds)
	assert.NotNil(t, file.ExpiresAt)
}

func TestFileRepo_Create_WithWorkflowID(t *testing.T) {
	repo, db, cleanup := setupFileRepoTest(t)
	defer cleanup()

	// Create workflow first
	workflowRepo := NewWorkflowRepository(db)
	workflow := &models.WorkflowModel{
		Name:      "Test Workflow",
		Status:    "active",
		Version:   1,
		Variables: models.JSONBMap{},
		Metadata:  models.JSONBMap{},
	}
	err := workflowRepo.Create(context.Background(), workflow)
	require.NoError(t, err)

	file := &models.FileModel{
		StorageID:  "local",
		Name:       "workflow-file.txt",
		Path:       "/workflow/file.txt",
		MimeType:   "text/plain",
		Size:       256,
		Checksum:   "def456",
		WorkflowID: &workflow.ID,
	}

	err = repo.Create(context.Background(), file)
	require.NoError(t, err)
	assert.Equal(t, workflow.ID, *file.WorkflowID)
}

// ========== UPDATE TESTS ==========

func TestFileRepo_Update_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	file := &models.FileModel{
		StorageID: "local",
		Name:      "original.txt",
		Path:      "/uploads/original.txt",
		MimeType:  "text/plain",
		Size:      100,
		Checksum:  "abc",
	}

	err := repo.Create(context.Background(), file)
	require.NoError(t, err)

	// Update file
	file.Size = 200
	file.Metadata = models.JSONBMap{"updated": true}

	err = repo.Update(context.Background(), file)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID(context.Background(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(200), found.Size)
	assert.Equal(t, true, found.Metadata["updated"])
}

// ========== DELETE TESTS ==========

func TestFileRepo_Delete_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	file := &models.FileModel{
		StorageID: "local",
		Name:      "delete-me.txt",
		Path:      "/temp/delete-me.txt",
		MimeType:  "text/plain",
		Size:      50,
		Checksum:  "hash",
	}

	err := repo.Create(context.Background(), file)
	require.NoError(t, err)

	// Delete file
	err = repo.Delete(context.Background(), file.ID)
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByID(context.Background(), file.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

// ========== FIND BY ID TESTS ==========

func TestFileRepo_FindByID_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	file := &models.FileModel{
		StorageID: "local",
		Name:      "find-me.txt",
		Path:      "/files/find-me.txt",
		MimeType:  "text/plain",
		Size:      1024,
		Checksum:  "findme",
	}

	err := repo.Create(context.Background(), file)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, file.Name, found.Name)
	assert.Equal(t, file.Path, found.Path)
}

func TestFileRepo_FindByID_NotFound(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	found, err := repo.FindByID(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, found)
}

// ========== FIND BY STORAGE AND PATH TESTS ==========

func TestFileRepo_FindByStorageAndPath_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	file := &models.FileModel{
		StorageID: "s3-bucket",
		Name:      "unique.txt",
		Path:      "/bucket/unique.txt",
		MimeType:  "text/plain",
		Size:      512,
		Checksum:  "unique",
	}

	err := repo.Create(context.Background(), file)
	require.NoError(t, err)

	found, err := repo.FindByStorageAndPath(context.Background(), "s3-bucket", "/bucket/unique.txt")
	require.NoError(t, err)
	assert.Equal(t, file.ID, found.ID)
	assert.Equal(t, file.Name, found.Name)
}

func TestFileRepo_FindByStorageAndPath_NotFound(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	found, err := repo.FindByStorageAndPath(context.Background(), "nonexistent", "/missing.txt")
	assert.Error(t, err)
	assert.Nil(t, found)
}

// ========== FIND BY QUERY TESTS ==========

func TestFileRepo_FindByQuery_ByStorageID(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	// Create files in different storages
	files := []*models.FileModel{
		{StorageID: "local", Name: "file1.txt", Path: "/local/1", MimeType: "text/plain", Size: 100, Checksum: "1"},
		{StorageID: "local", Name: "file2.txt", Path: "/local/2", MimeType: "text/plain", Size: 200, Checksum: "2"},
		{StorageID: "s3", Name: "file3.txt", Path: "/s3/3", MimeType: "text/plain", Size: 300, Checksum: "3"},
	}

	for _, f := range files {
		err := repo.Create(context.Background(), f)
		require.NoError(t, err)
	}

	query := &FileQuery{StorageID: "local"}
	result, err := repo.FindByQuery(context.Background(), query)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), 2)
}

func TestFileRepo_FindByQuery_ByTags(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	// Create file with tags
	file := &models.FileModel{
		StorageID: "local",
		Name:      "tagged.txt",
		Path:      "/tagged/file.txt",
		MimeType:  "text/plain",
		Size:      100,
		Checksum:  "tagged",
		Tags:      models.StringArray{"important", "document"},
	}

	err := repo.Create(context.Background(), file)
	require.NoError(t, err)

	query := &FileQuery{Tags: []string{"important"}}
	result, err := repo.FindByQuery(context.Background(), query)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), 1)
}

func TestFileRepo_FindByQuery_Pagination(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	// Create multiple files
	for i := 0; i < 5; i++ {
		file := &models.FileModel{
			StorageID: "local",
			Name:      fmt.Sprintf("file%d.txt", i),
			Path:      fmt.Sprintf("/files/%d.txt", i),
			MimeType:  "text/plain",
			Size:      int64(i * 100),
			Checksum:  fmt.Sprintf("hash%d", i),
		}
		err := repo.Create(context.Background(), file)
		require.NoError(t, err)
	}

	// Get first page
	query := &FileQuery{StorageID: "local", Limit: 2, Offset: 0}
	page1, err := repo.FindByQuery(context.Background(), query)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get second page
	query.Offset = 2
	page2, err := repo.FindByQuery(context.Background(), query)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify different files
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

// ========== COUNT TESTS ==========

func TestFileRepo_CountByQuery_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	// Create files
	for i := 0; i < 3; i++ {
		file := &models.FileModel{
			StorageID: "local",
			Name:      fmt.Sprintf("count%d.txt", i),
			Path:      fmt.Sprintf("/count/%d.txt", i),
			MimeType:  "text/plain",
			Size:      100,
			Checksum:  fmt.Sprintf("c%d", i),
		}
		err := repo.Create(context.Background(), file)
		require.NoError(t, err)
	}

	query := &FileQuery{StorageID: "local"}
	count, err := repo.CountByQuery(context.Background(), query)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 3)
}

// ========== FIND EXPIRED TESTS ==========

func TestFileRepo_FindExpired_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	// Create expired file
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredFile := &models.FileModel{
		StorageID: "local",
		Name:      "expired.txt",
		Path:      "/temp/expired.txt",
		MimeType:  "text/plain",
		Size:      100,
		Checksum:  "expired",
		ExpiresAt: &expiredTime,
	}

	err := repo.Create(context.Background(), expiredFile)
	require.NoError(t, err)

	// Create non-expired file
	futureTime := time.Now().Add(1 * time.Hour)
	activeFile := &models.FileModel{
		StorageID: "local",
		Name:      "active.txt",
		Path:      "/files/active.txt",
		MimeType:  "text/plain",
		Size:      100,
		Checksum:  "active",
		ExpiresAt: &futureTime,
	}

	err = repo.Create(context.Background(), activeFile)
	require.NoError(t, err)

	// Find expired files
	expired, err := repo.FindExpired(context.Background(), 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(expired), 1)

	// Verify expired file is in the list
	found := false
	for _, f := range expired {
		if f.ID == expiredFile.ID {
			found = true
			break
		}
	}
	assert.True(t, found)
}

// ========== STORAGE USAGE TESTS ==========

func TestFileRepo_GetStorageUsage_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	storageID := "usage-test"

	// Create files
	totalSize := int64(0)
	for i := 0; i < 3; i++ {
		size := int64((i + 1) * 100)
		file := &models.FileModel{
			StorageID: storageID,
			Name:      fmt.Sprintf("file%d.txt", i),
			Path:      fmt.Sprintf("/usage/%d.txt", i),
			MimeType:  "text/plain",
			Size:      size,
			Checksum:  fmt.Sprintf("u%d", i),
		}
		err := repo.Create(context.Background(), file)
		require.NoError(t, err)
		totalSize += size
	}

	size, count, err := repo.GetStorageUsage(context.Background(), storageID)
	require.NoError(t, err)
	assert.Equal(t, totalSize, size)
	assert.Equal(t, int64(3), count)
}

func TestFileRepo_GetStorageUsage_EmptyStorage(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	size, count, err := repo.GetStorageUsage(context.Background(), "empty-storage")
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)
	assert.Equal(t, int64(0), count)
}

// ========== DELETE BY STORAGE ID TESTS ==========

func TestFileRepo_DeleteByStorageID_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	storageID := "delete-storage"

	// Create files in storage
	for i := 0; i < 3; i++ {
		file := &models.FileModel{
			StorageID: storageID,
			Name:      fmt.Sprintf("del%d.txt", i),
			Path:      fmt.Sprintf("/del/%d.txt", i),
			MimeType:  "text/plain",
			Size:      100,
			Checksum:  fmt.Sprintf("d%d", i),
		}
		err := repo.Create(context.Background(), file)
		require.NoError(t, err)
	}

	// Delete all files in storage
	deleted, err := repo.DeleteByStorageID(context.Background(), storageID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)

	// Verify files are deleted
	query := &FileQuery{StorageID: storageID}
	remaining, err := repo.FindByQuery(context.Background(), query)
	require.NoError(t, err)
	assert.Len(t, remaining, 0)
}

// ========== DELETE EXPIRED TESTS ==========

func TestFileRepo_DeleteExpired_Success(t *testing.T) {
	repo, _, cleanup := setupFileRepoTest(t)
	defer cleanup()

	// Create expired files
	expiredTime := time.Now().Add(-1 * time.Hour)
	for i := 0; i < 2; i++ {
		file := &models.FileModel{
			StorageID: "temp",
			Name:      fmt.Sprintf("expired%d.txt", i),
			Path:      fmt.Sprintf("/temp/expired%d.txt", i),
			MimeType:  "text/plain",
			Size:      100,
			Checksum:  fmt.Sprintf("exp%d", i),
			ExpiresAt: &expiredTime,
		}
		err := repo.Create(context.Background(), file)
		require.NoError(t, err)
	}

	// Create active file
	futureTime := time.Now().Add(1 * time.Hour)
	activeFile := &models.FileModel{
		StorageID: "temp",
		Name:      "active.txt",
		Path:      "/temp/active.txt",
		MimeType:  "text/plain",
		Size:      100,
		Checksum:  "act",
		ExpiresAt: &futureTime,
	}
	err := repo.Create(context.Background(), activeFile)
	require.NoError(t, err)

	// Delete expired files
	deleted, err := repo.DeleteExpired(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, int64(2))

	// Verify active file still exists
	found, err := repo.FindByID(context.Background(), activeFile.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
}

// ========== FILE MODEL HELPER TESTS ==========

func TestFileModel_IsExpired(t *testing.T) {
	// Expired file
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredFile := &models.FileModel{
		ExpiresAt: &expiredTime,
	}
	assert.True(t, expiredFile.IsExpired())

	// Active file
	futureTime := time.Now().Add(1 * time.Hour)
	activeFile := &models.FileModel{
		ExpiresAt: &futureTime,
	}
	assert.False(t, activeFile.IsExpired())

	// No expiration
	noExpireFile := &models.FileModel{
		ExpiresAt: nil,
	}
	assert.False(t, noExpireFile.IsExpired())
}
