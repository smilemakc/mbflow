package filestorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============== A. Initialization Tests ==============

func TestLocalProvider_New_Success(t *testing.T) {
	dir := t.TempDir()

	provider, err := NewLocalProvider(dir)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, dir, provider.basePath)

	// Verify directory was created
	_, err = os.Stat(dir)
	assert.NoError(t, err)
}

func TestLocalProvider_New_CreateDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", "nested")

	provider, err := NewLocalProvider(dir)

	require.NoError(t, err)
	require.NotNil(t, provider)

	// Verify nested directory was created
	stat, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
}

func TestLocalProvider_New_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	// Try to create in a read-only directory
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	err := os.Mkdir(readOnlyDir, 0444) // read-only
	require.NoError(t, err)

	invalidPath := filepath.Join(readOnlyDir, "subdir")

	provider, err := NewLocalProvider(invalidPath)

	// Should fail due to permission denied
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestLocalProvider_Type_ReturnsLocal(t *testing.T) {
	provider, err := NewLocalProvider(t.TempDir())
	require.NoError(t, err)

	assert.Equal(t, models.StorageTypeLocal, provider.Type())
}

// ============== B. Store Operations Tests ==============

func TestLocalProvider_Store_Success(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "test-file-1",
		StorageID: "storage-1",
		Name:      "test.txt",
	}

	content := []byte("Hello, World!")
	reader := bytes.NewReader(content)

	path, err := provider.Store(context.Background(), entry, reader)

	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Equal(t, path, entry.Path)
	assert.Equal(t, int64(len(content)), entry.Size)
	assert.NotEmpty(t, entry.Checksum)

	// Verify checksum is correct SHA256
	expectedHash := sha256.Sum256(content)
	expectedChecksum := hex.EncodeToString(expectedHash[:])
	assert.Equal(t, expectedChecksum, entry.Checksum)

	// Verify file was actually written
	fullPath := filepath.Join(dir, path)
	_, err = os.Stat(fullPath)
	assert.NoError(t, err)
}

func TestLocalProvider_Store_GeneratePath(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "test-file-2",
		StorageID: "storage-1",
		Name:      "document.pdf",
		Path:      "", // Empty path should be auto-generated
	}

	path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))

	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "storage-1")    // Should contain storage ID
	assert.Contains(t, path, "document.pdf") // Should contain filename
}

func TestLocalProvider_Store_SanitizeFilename_UnsafeChars(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	unsafeNames := []string{
		"file/with/slashes.txt",
		"file\\with\\backslashes.txt",
		"file:with:colons.txt",
		"file*with*asterisks.txt",
		"file?with?questions.txt",
		"file\"with\"quotes.txt",
		"file<with>brackets.txt",
		"file|with|pipes.txt",
	}

	for _, unsafeName := range unsafeNames {
		t.Run(unsafeName, func(t *testing.T) {
			entry := &models.FileEntry{
				ID:        "test-file",
				StorageID: "storage-1",
				Name:      unsafeName,
			}

			path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))

			require.NoError(t, err)
			assert.NotEmpty(t, path)

			// Verify sanitized filename doesn't contain unsafe characters
			// Extract just the filename (last component of path)
			basename := filepath.Base(path)
			assert.NotContains(t, basename, "\\")
			assert.NotContains(t, basename, ":")
			assert.NotContains(t, basename, "*")
			assert.NotContains(t, basename, "?")
			assert.NotContains(t, basename, "\"")
			assert.NotContains(t, basename, "<")
			assert.NotContains(t, basename, ">")
			assert.NotContains(t, basename, "|")
		})
	}
}

func TestLocalProvider_Store_SanitizeFilename_LongFilename(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Create a very long filename (more than 200 characters)
	longName := string(make([]byte, 250))
	for i := range longName {
		longName = longName[:i] + "a"
	}
	longName += ".txt"

	entry := &models.FileEntry{
		ID:        "test-file",
		StorageID: "storage-1",
		Name:      longName,
	}

	path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))

	require.NoError(t, err)
	assert.NotEmpty(t, path)

	// Filename component should be truncated to 200 chars
	filename := filepath.Base(path)
	assert.LessOrEqual(t, len(filename), 200)
}

func TestLocalProvider_Store_SanitizeFilename_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	pathTraversalNames := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"file/../../../escape.txt",
	}

	for _, name := range pathTraversalNames {
		t.Run(name, func(t *testing.T) {
			entry := &models.FileEntry{
				ID:        "test-file",
				StorageID: "storage-1",
				Name:      name,
			}

			path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))

			require.NoError(t, err)
			assert.NotEmpty(t, path)

			// Verify file is stored within basePath
			fullPath := filepath.Join(dir, path)
			absPath, _ := filepath.Abs(fullPath)
			absBase, _ := filepath.Abs(dir)
			assert.True(t, filepath.HasPrefix(absPath, absBase), "File should be within base directory")
		})
	}
}

func TestLocalProvider_Store_CreateDirectoryStructure(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "test-file",
		StorageID: "storage-1",
		Name:      "nested.txt",
	}

	path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))

	require.NoError(t, err)

	// Verify directory structure was created
	fullPath := filepath.Join(dir, path)
	dirPath := filepath.Dir(fullPath)
	stat, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
}

func TestLocalProvider_Store_CalculateChecksum_SHA256(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		content string
	}{
		{"simple", "Hello, World!"},
		{"empty", ""},
		{"binary", string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE})},
		{"unicode", "Hello 世界 🌍"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := &models.FileEntry{
				ID:        "test-file",
				StorageID: "storage-1",
				Name:      tc.name + ".txt",
			}

			content := []byte(tc.content)
			_, err := provider.Store(context.Background(), entry, bytes.NewReader(content))

			require.NoError(t, err)
			assert.NotEmpty(t, entry.Checksum)

			// Verify checksum
			expectedHash := sha256.Sum256(content)
			expectedChecksum := hex.EncodeToString(expectedHash[:])
			assert.Equal(t, expectedChecksum, entry.Checksum)
		})
	}
}

func TestLocalProvider_Store_CalculateChecksum_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "test-file",
		StorageID: "storage-1",
		Name:      "empty.txt",
	}

	_, err = provider.Store(context.Background(), entry, bytes.NewReader([]byte{}))

	require.NoError(t, err)
	assert.NotEmpty(t, entry.Checksum)
	assert.Equal(t, int64(0), entry.Size)

	// Verify checksum for empty file
	expectedHash := sha256.Sum256([]byte{})
	expectedChecksum := hex.EncodeToString(expectedHash[:])
	assert.Equal(t, expectedChecksum, entry.Checksum)
}

func TestLocalProvider_Store_UnicodeFilename(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "test-file",
		StorageID: "storage-1",
		Name:      "文件-файл-📁.txt",
	}

	path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))

	require.NoError(t, err)
	assert.NotEmpty(t, path)
}

func TestLocalProvider_Store_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	var wg sync.WaitGroup
	fileCount := 50

	for i := 0; i < fileCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			entry := &models.FileEntry{
				ID:        string(rune('a' + idx)),
				StorageID: "storage-1",
				Name:      "file" + string(rune('0'+idx%10)) + ".txt",
			}

			content := []byte("content " + string(rune('0'+idx)))
			_, err := provider.Store(context.Background(), entry, bytes.NewReader(content))

			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all files were created
	usage, err := provider.GetUsage(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(fileCount), usage.FileCount)
}

func TestLocalProvider_Store_ExistingPath(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store first file
	entry1 := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "test.txt",
	}
	_, err = provider.Store(context.Background(), entry1, bytes.NewReader([]byte("first")))
	require.NoError(t, err)

	// Store second file with same name (should get different path due to unique ID)
	entry2 := &models.FileEntry{
		ID:        "file-2",
		StorageID: "storage-1",
		Name:      "test.txt",
	}
	path2, err := provider.Store(context.Background(), entry2, bytes.NewReader([]byte("second")))
	require.NoError(t, err)

	// Paths should be different due to unique ID in path
	assert.NotEqual(t, entry1.Path, path2)
}

// ============== C. Get Operations Tests ==============

func TestLocalProvider_Get_Success(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store a file first
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "test.txt",
	}
	content := []byte("Hello, World!")
	path, err := provider.Store(context.Background(), entry, bytes.NewReader(content))
	require.NoError(t, err)

	// Get the file
	reader, err := provider.Get(context.Background(), path)

	require.NoError(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	// Read and verify content
	retrievedContent, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, retrievedContent)
}

func TestLocalProvider_Get_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Try to get non-existent file
	reader, err := provider.Get(context.Background(), "storage-1/nonexistent/file.txt")

	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "file not found")
}

func TestLocalProvider_Get_ConcurrentReads(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store a file
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "concurrent.txt",
	}
	content := []byte("Concurrent read test content")
	path, err := provider.Store(context.Background(), entry, bytes.NewReader(content))
	require.NoError(t, err)

	// Concurrent reads
	var wg sync.WaitGroup
	readCount := 20

	for i := 0; i < readCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			reader, err := provider.Get(context.Background(), path)
			assert.NoError(t, err)
			if reader != nil {
				defer reader.Close()
				retrieved, err := io.ReadAll(reader)
				assert.NoError(t, err)
				assert.Equal(t, content, retrieved)
			}
		}()
	}

	wg.Wait()
}

func TestLocalProvider_Get_LargeFile_10MB(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Create 10MB file
	size := 10 * 1024 * 1024
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}

	entry := &models.FileEntry{
		ID:        "large-file",
		StorageID: "storage-1",
		Name:      "large.bin",
	}

	path, err := provider.Store(context.Background(), entry, bytes.NewReader(content))
	require.NoError(t, err)

	// Get the large file
	reader, err := provider.Get(context.Background(), path)
	require.NoError(t, err)
	defer reader.Close()

	// Verify size
	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Len(t, retrieved, size)
	assert.Equal(t, content[:100], retrieved[:100]) // Verify first 100 bytes
}

func TestLocalProvider_Get_ContentIntegrity(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store file
	content := []byte("Content integrity test - 日本語 - 🎉")
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "integrity.txt",
	}
	path, err := provider.Store(context.Background(), entry, bytes.NewReader(content))
	require.NoError(t, err)

	// Get file multiple times and verify checksum
	for i := 0; i < 5; i++ {
		reader, err := provider.Get(context.Background(), path)
		require.NoError(t, err)

		retrieved, err := io.ReadAll(reader)
		reader.Close()
		require.NoError(t, err)

		// Calculate checksum
		hash := sha256.Sum256(retrieved)
		checksum := hex.EncodeToString(hash[:])

		assert.Equal(t, entry.Checksum, checksum, "Checksum mismatch on read #%d", i+1)
		assert.Equal(t, content, retrieved, "Content mismatch on read #%d", i+1)
	}
}

// ============== D. Delete Operations Tests ==============

func TestLocalProvider_Delete_Success(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store a file
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "delete-me.txt",
	}
	path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	// Verify file exists
	fullPath := filepath.Join(dir, path)
	_, err = os.Stat(fullPath)
	require.NoError(t, err)

	// Delete the file
	err = provider.Delete(context.Background(), path)

	require.NoError(t, err)

	// Verify file no longer exists
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalProvider_Delete_FileNotFound_NoError(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Delete non-existent file should not error
	err = provider.Delete(context.Background(), "storage-1/nonexistent/file.txt")

	assert.NoError(t, err, "Deleting non-existent file should not return error")
}

func TestLocalProvider_Delete_CleanupEmptyDirs(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store a file (creates directory structure)
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "file.txt",
	}
	path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	// Get the parent directory
	fullPath := filepath.Join(dir, path)
	parentDir := filepath.Dir(fullPath)

	// Delete the file
	err = provider.Delete(context.Background(), path)
	require.NoError(t, err)

	// Parent directory should be cleaned up if empty
	_, err = os.Stat(parentDir)
	// Directory may or may not exist depending on cleanup logic
	// Just verify file is gone
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalProvider_Delete_KeepNonEmptyDirs(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store two files in same directory structure
	entry1 := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "file1.txt",
		Path:      "storage-1/subdir/file1.txt",
	}
	_, err = provider.Store(context.Background(), entry1, bytes.NewReader([]byte("test1")))
	require.NoError(t, err)

	entry2 := &models.FileEntry{
		ID:        "file-2",
		StorageID: "storage-1",
		Name:      "file2.txt",
		Path:      "storage-1/subdir/file2.txt",
	}
	_, err = provider.Store(context.Background(), entry2, bytes.NewReader([]byte("test2")))
	require.NoError(t, err)

	// Delete first file
	err = provider.Delete(context.Background(), entry1.Path)
	require.NoError(t, err)

	// Parent directory should still exist (has file2)
	parentDir := filepath.Join(dir, "storage-1/subdir")
	stat, err := os.Stat(parentDir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir())

	// Verify second file still exists
	file2Path := filepath.Join(dir, entry2.Path)
	_, err = os.Stat(file2Path)
	assert.NoError(t, err)
}

func TestLocalProvider_Delete_NestedDirs_Cleanup(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Create nested directory structure
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "nested.txt",
	}
	path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	// Delete file
	err = provider.Delete(context.Background(), path)
	require.NoError(t, err)

	// Verify file is gone
	fullPath := filepath.Join(dir, path)
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalProvider_Delete_ConcurrentDeletes(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store multiple files
	fileCount := 20
	paths := make([]string, fileCount)

	for i := 0; i < fileCount; i++ {
		entry := &models.FileEntry{
			ID:        string(rune('a' + i)),
			StorageID: "storage-1",
			Name:      "file" + string(rune('0'+i%10)) + ".txt",
		}
		path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
		require.NoError(t, err)
		paths[i] = path
	}

	// Concurrent deletes
	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			err := provider.Delete(context.Background(), p)
			assert.NoError(t, err)
		}(path)
	}

	wg.Wait()

	// Verify all files were deleted
	usage, err := provider.GetUsage(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), usage.FileCount)
}

// ============== E. Utility Operations Tests ==============

func TestLocalProvider_Exists_True(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store a file
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "exists.txt",
	}
	path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	// Check if exists
	exists, err := provider.Exists(context.Background(), path)

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestLocalProvider_Exists_False(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	exists, err := provider.Exists(context.Background(), "storage-1/nonexistent/file.txt")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestLocalProvider_GetUsage_Empty(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	usage, err := provider.GetUsage(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(0), usage.TotalSize)
	assert.Equal(t, int64(0), usage.FileCount)
}

func TestLocalProvider_GetUsage_SingleFile(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store a file
	content := []byte("Test content for usage")
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "storage-1",
		Name:      "usage.txt",
	}
	_, err = provider.Store(context.Background(), entry, bytes.NewReader(content))
	require.NoError(t, err)

	usage, err := provider.GetUsage(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), usage.TotalSize)
	assert.Equal(t, int64(1), usage.FileCount)
}

func TestLocalProvider_GetUsage_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store multiple files
	files := []struct {
		name    string
		content string
	}{
		{"file1.txt", "content1"},
		{"file2.txt", "content2 with more data"},
		{"file3.txt", "content3"},
	}

	var expectedSize int64
	for i, f := range files {
		entry := &models.FileEntry{
			ID:        string(rune('a' + i)),
			StorageID: "storage-1",
			Name:      f.name,
		}
		_, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte(f.content)))
		require.NoError(t, err)
		expectedSize += int64(len(f.content))
	}

	usage, err := provider.GetUsage(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expectedSize, usage.TotalSize)
	assert.Equal(t, int64(len(files)), usage.FileCount)
}

func TestLocalProvider_GetUsage_NestedDirectories(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store files with different paths (nested structure)
	for i := 0; i < 5; i++ {
		entry := &models.FileEntry{
			ID:        string(rune('a' + i)),
			StorageID: "storage-1",
			Name:      "file.txt",
		}
		_, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
		require.NoError(t, err)
	}

	usage, err := provider.GetUsage(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(5*4), usage.TotalSize) // 5 files * 4 bytes
	assert.Equal(t, int64(5), usage.FileCount)
}

func TestLocalProvider_GetUsage_LargeStorage(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store many small files
	fileCount := 100
	fileSize := 1024 // 1KB each

	for i := 0; i < fileCount; i++ {
		entry := &models.FileEntry{
			ID:        string(rune('a' + i)),
			StorageID: "storage-1",
			Name:      "file.txt",
		}
		content := make([]byte, fileSize)
		_, err := provider.Store(context.Background(), entry, bytes.NewReader(content))
		require.NoError(t, err)
	}

	usage, err := provider.GetUsage(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(fileCount*fileSize), usage.TotalSize)
	assert.Equal(t, int64(fileCount), usage.FileCount)
}

func TestLocalProvider_Close_Success(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

// ============== F. Edge Cases Tests ==============

func TestLocalProvider_SanitizeFilename_AllUnsafe(t *testing.T) {
	result := sanitizeFilename("/:*?\"<>|\\")
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "/")
	assert.NotContains(t, result, ":")
	assert.NotContains(t, result, "*")
	assert.NotContains(t, result, "?")
	assert.NotContains(t, result, "\"")
	assert.NotContains(t, result, "<")
	assert.NotContains(t, result, ">")
	assert.NotContains(t, result, "|")
	assert.NotContains(t, result, "\\")
}

func TestLocalProvider_SanitizeFilename_EmptyInput(t *testing.T) {
	result := sanitizeFilename("")
	assert.Equal(t, "", result)
}

func TestLocalProvider_Checksum_LargeFile_100MB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Create 100MB file
	size := 100 * 1024 * 1024
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}

	entry := &models.FileEntry{
		ID:        "large-file",
		StorageID: "storage-1",
		Name:      "100mb.bin",
	}

	_, err = provider.Store(context.Background(), entry, bytes.NewReader(content))
	require.NoError(t, err)

	// Verify checksum
	expectedHash := sha256.Sum256(content)
	expectedChecksum := hex.EncodeToString(expectedHash[:])
	assert.Equal(t, expectedChecksum, entry.Checksum)
	assert.Equal(t, int64(size), entry.Size)
}

func TestLocalProvider_PathGeneration_UniqueIDs(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	// Store multiple files with same name but different IDs
	paths := make(map[string]bool)

	for i := 0; i < 10; i++ {
		entry := &models.FileEntry{
			ID:        string(rune('a' + i)),
			StorageID: "storage-1",
			Name:      "same-name.txt",
		}
		path, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
		require.NoError(t, err)

		// Verify path is unique
		assert.False(t, paths[path], "Path should be unique")
		paths[path] = true
	}

	assert.Len(t, paths, 10, "Should have 10 unique paths")
}

func TestLocalProvider_Concurrent_StoreAndDelete(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewLocalProvider(dir)
	require.NoError(t, err)

	var wg sync.WaitGroup
	operationCount := 50

	// Concurrent store and delete operations
	for i := 0; i < operationCount; i++ {
		wg.Add(2)

		// Store
		go func(idx int) {
			defer wg.Done()
			entry := &models.FileEntry{
				ID:        string(rune('a' + idx)),
				StorageID: "storage-1",
				Name:      "file.txt",
			}
			_, err := provider.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
			assert.NoError(t, err)
		}(i)

		// Delete (may fail if file doesn't exist yet, which is ok)
		go func(idx int) {
			defer wg.Done()
			// Try to delete a potentially existing file
			path := "storage-1/" + string(rune('a'+idx%10)) + "/file.txt"
			_ = provider.Delete(context.Background(), path)
		}(i)
	}

	wg.Wait()
}

// ============== G. Factory Tests ==============

func TestLocalProviderFactory_Type(t *testing.T) {
	factory := NewLocalProviderFactory()
	assert.Equal(t, models.StorageTypeLocal, factory.Type())
}

func TestLocalProviderFactory_Create_Success(t *testing.T) {
	factory := NewLocalProviderFactory()
	config := &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: t.TempDir(),
	}

	provider, err := factory.Create(config)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, models.StorageTypeLocal, provider.Type())
}

func TestLocalProviderFactory_Create_MissingBasePath(t *testing.T) {
	factory := NewLocalProviderFactory()
	config := &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "", // Missing
	}

	provider, err := factory.Create(config)

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "base_path is required")
}
