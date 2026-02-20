package rest

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// FileHandlers provides HTTP handlers for file storage endpoints
type FileHandlers struct {
	fileRepo       *storage.FileRepository
	storageManager filestorage.Manager
	logger         *logger.Logger
}

// NewFileHandlers creates a new FileHandlers instance
func NewFileHandlers(fileRepo *storage.FileRepository, storageManager filestorage.Manager, log *logger.Logger) *FileHandlers {
	return &FileHandlers{
		fileRepo:       fileRepo,
		storageManager: storageManager,
		logger:         log,
	}
}

// HandleUploadFile handles POST /api/v1/files
// Accepts multipart/form-data or JSON with base64 data
func (h *FileHandlers) HandleUploadFile(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.handleMultipartUpload(c)
	} else {
		h.handleJSONUpload(c)
	}
}

// handleMultipartUpload handles multipart/form-data file upload
func (h *FileHandlers) handleMultipartUpload(c *gin.Context) {
	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Get form values
	storageID := c.PostForm("storage_id")
	if storageID == "" {
		storageID = "default"
	}
	accessScope := c.PostForm("access_scope")
	if accessScope == "" {
		accessScope = "workflow"
	}
	workflowID := c.PostForm("workflow_id")
	executionID := c.PostForm("execution_id")
	tagsStr := c.PostForm("tags")

	// Parse tags
	var tags []string
	if tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Detect MIME type
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])
	file.Seek(0, 0) // Reset reader

	// Validate MIME type
	if !models.IsMimeTypeAllowed(mimeType) {
		respondError(c, http.StatusBadRequest, fmt.Sprintf("MIME type not allowed: %s", mimeType))
		return
	}

	// Get storage
	store, err := h.storageManager.GetStorage(storageID)
	if err != nil {
		h.logger.Error("Failed to get storage", "error", err, "storage_id", storageID)
		respondError(c, http.StatusInternalServerError, "failed to get storage")
		return
	}

	// Create file entry
	entry := &models.FileEntry{
		StorageID:   storageID,
		Name:        header.Filename,
		MimeType:    mimeType,
		Size:        header.Size,
		AccessScope: models.AccessScope(accessScope),
		Tags:        tags,
		Metadata:    make(map[string]any),
	}

	// Set optional workflow/execution
	if workflowID != "" {
		entry.WorkflowID = &workflowID
	}
	if executionID != "" {
		entry.ExecutionID = &executionID
	}

	// Store file
	stored, err := store.Store(c.Request.Context(), entry, file)
	if err != nil {
		h.logger.Error("Failed to store file", "error", err, "filename", header.Filename)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Save to database
	fileModel := h.entryToModel(stored)
	if err := h.fileRepo.Create(c.Request.Context(), fileModel); err != nil {
		h.logger.Error("Failed to save file metadata", "error", err, "file_id", stored.ID)
		respondError(c, http.StatusInternalServerError, "failed to save file metadata")
		return
	}

	respondJSON(c, http.StatusCreated, h.modelToResponse(fileModel))
}

// handleJSONUpload handles JSON body with base64 file data
func (h *FileHandlers) handleJSONUpload(c *gin.Context) {
	var req struct {
		StorageID   string   `json:"storage_id"`
		FileName    string   `json:"file_name"`
		FileData    string   `json:"file_data"` // Base64
		FileURL     string   `json:"file_url"`
		MimeType    string   `json:"mime_type"`
		AccessScope string   `json:"access_scope"`
		Tags        []string `json:"tags"`
		WorkflowID  string   `json:"workflow_id"`
		ExecutionID string   `json:"execution_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate input
	if req.FileData == "" && req.FileURL == "" {
		respondError(c, http.StatusBadRequest, "either file_data or file_url is required")
		return
	}

	if req.FileName == "" {
		req.FileName = fmt.Sprintf("file_%s", uuid.New().String()[:8])
	}

	storageID := req.StorageID
	if storageID == "" {
		storageID = "default"
	}

	accessScope := req.AccessScope
	if accessScope == "" {
		accessScope = "workflow"
	}

	// Get file data
	var reader io.Reader
	var size int64

	if req.FileData != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.FileData)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid base64 data")
			return
		}
		reader = strings.NewReader(string(decoded))
		size = int64(len(decoded))

		// Detect MIME type if not provided
		if req.MimeType == "" {
			req.MimeType = http.DetectContentType(decoded[:min(512, len(decoded))])
		}
	} else {
		// Download from URL
		resp, err := http.Get(req.FileURL)
		if err != nil {
			respondError(c, http.StatusBadRequest, "failed to download file from URL")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("failed to download file: HTTP %d", resp.StatusCode))
			return
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			respondError(c, http.StatusBadRequest, "failed to read file from URL")
			return
		}
		reader = strings.NewReader(string(data))
		size = int64(len(data))

		if req.MimeType == "" {
			req.MimeType = resp.Header.Get("Content-Type")
			if req.MimeType == "" {
				req.MimeType = http.DetectContentType(data[:min(512, len(data))])
			}
		}
	}

	// Validate MIME type
	if !models.IsMimeTypeAllowed(req.MimeType) {
		respondError(c, http.StatusBadRequest, fmt.Sprintf("MIME type not allowed: %s", req.MimeType))
		return
	}

	// Get storage
	store, err := h.storageManager.GetStorage(storageID)
	if err != nil {
		h.logger.Error("Failed to get storage", "error", err, "storage_id", storageID)
		respondError(c, http.StatusInternalServerError, "failed to get storage")
		return
	}

	// Create file entry
	entry := &models.FileEntry{
		StorageID:   storageID,
		Name:        req.FileName,
		MimeType:    req.MimeType,
		Size:        size,
		AccessScope: models.AccessScope(accessScope),
		Tags:        req.Tags,
		Metadata:    make(map[string]any),
	}

	if req.WorkflowID != "" {
		entry.WorkflowID = &req.WorkflowID
	}
	if req.ExecutionID != "" {
		entry.ExecutionID = &req.ExecutionID
	}

	// Store file
	stored, err := store.Store(c.Request.Context(), entry, reader)
	if err != nil {
		h.logger.Error("Failed to store file", "error", err, "filename", req.FileName)
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Save to database
	fileModel := h.entryToModel(stored)
	if err := h.fileRepo.Create(c.Request.Context(), fileModel); err != nil {
		h.logger.Error("Failed to save file metadata", "error", err, "file_id", stored.ID)
		respondError(c, http.StatusInternalServerError, "failed to save file metadata")
		return
	}

	respondJSON(c, http.StatusCreated, h.modelToResponse(fileModel))
}

// HandleGetFile handles GET /api/v1/files/:id
func (h *FileHandlers) HandleGetFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		respondError(c, http.StatusBadRequest, "file ID is required")
		return
	}

	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid file ID")
		return
	}

	// Get file metadata
	fileModel, err := h.fileRepo.FindByID(c.Request.Context(), fileUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "file not found")
		return
	}

	// Check if expired
	if fileModel.IsExpired() {
		respondError(c, http.StatusGone, "file has expired")
		return
	}

	// Get storage
	store, err := h.storageManager.GetStorage(fileModel.StorageID)
	if err != nil {
		h.logger.Error("Failed to get storage", "error", err, "storage_id", fileModel.StorageID)
		respondError(c, http.StatusInternalServerError, "storage not available")
		return
	}

	// Get file content using the stored path
	_, reader, err := store.Get(c.Request.Context(), fileModel.Path)
	if err != nil {
		h.logger.Error("Failed to get file content", "error", err, "file_id", fileID)
		respondError(c, http.StatusInternalServerError, "failed to retrieve file")
		return
	}
	defer reader.Close()

	// Set headers for download using metadata from DB
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileModel.Name))
	c.Header("Content-Type", fileModel.MimeType)

	// Stream file
	c.DataFromReader(http.StatusOK, fileModel.Size, fileModel.MimeType, reader, nil)
}

// HandleGetFileMetadata handles GET /api/v1/files/:id/metadata
func (h *FileHandlers) HandleGetFileMetadata(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		respondError(c, http.StatusBadRequest, "file ID is required")
		return
	}

	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid file ID")
		return
	}

	fileModel, err := h.fileRepo.FindByID(c.Request.Context(), fileUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "file not found")
		return
	}

	respondJSON(c, http.StatusOK, h.modelToResponse(fileModel))
}

// HandleDeleteFile handles DELETE /api/v1/files/:id
func (h *FileHandlers) HandleDeleteFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		respondError(c, http.StatusBadRequest, "file ID is required")
		return
	}

	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid file ID")
		return
	}

	// Get file to get storage info
	fileModel, err := h.fileRepo.FindByID(c.Request.Context(), fileUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "file not found")
		return
	}

	// Delete from storage
	store, err := h.storageManager.GetStorage(fileModel.StorageID)
	if err == nil {
		_ = store.Delete(c.Request.Context(), fileID)
	}

	// Delete from database
	if err := h.fileRepo.Delete(c.Request.Context(), fileUUID); err != nil {
		h.logger.Error("Failed to delete file", "error", err, "file_id", fileID)
		respondError(c, http.StatusInternalServerError, "failed to delete file")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}

// HandleListFiles handles GET /api/v1/files
func (h *FileHandlers) HandleListFiles(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	query := &storage.FileQuery{
		StorageID:   c.Query("storage_id"),
		AccessScope: c.Query("access_scope"),
		Limit:       limit,
		Offset:      offset,
		OrderBy:     c.DefaultQuery("order_by", "created_at"),
		OrderDir:    c.DefaultQuery("order_dir", "DESC"),
	}

	// Parse workflow_id if provided
	if wfID := c.Query("workflow_id"); wfID != "" {
		if wfUUID, err := uuid.Parse(wfID); err == nil {
			query.WorkflowID = &wfUUID
		}
	}

	// Parse execution_id if provided
	if exID := c.Query("execution_id"); exID != "" {
		if exUUID, err := uuid.Parse(exID); err == nil {
			query.ExecutionID = &exUUID
		}
	}

	// Parse tags
	if tagsStr := c.Query("tags"); tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				query.Tags = append(query.Tags, tag)
			}
		}
	}

	// Parse MIME types
	if mimeTypes := c.Query("mime_types"); mimeTypes != "" {
		for _, mt := range strings.Split(mimeTypes, ",") {
			mt = strings.TrimSpace(mt)
			if mt != "" {
				query.MimeTypes = append(query.MimeTypes, mt)
			}
		}
	}

	files, err := h.fileRepo.FindByQuery(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("Failed to list files", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to list files")
		return
	}

	// Count total
	total, _ := h.fileRepo.CountByQuery(c.Request.Context(), query)

	// Convert to response format
	response := make([]map[string]any, len(files))
	for i, f := range files {
		response[i] = h.modelToResponse(f)
	}

	c.JSON(http.StatusOK, gin.H{
		"files":  response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// HandleGetStorageUsage handles GET /api/v1/files/storage/:storage_id/usage
func (h *FileHandlers) HandleGetStorageUsage(c *gin.Context) {
	storageID := c.Param("storage_id")
	if storageID == "" {
		storageID = "default"
	}

	accessScope := c.Query("access_scope")

	// If access_scope is provided, use filtered query
	if accessScope != "" {
		query := &storage.FileQuery{
			StorageID:   storageID,
			AccessScope: accessScope,
		}
		files, err := h.fileRepo.FindByQuery(c.Request.Context(), query)
		if err != nil {
			h.logger.Error("Failed to get storage usage", "error", err, "storage_id", storageID, "access_scope", accessScope)
			respondError(c, http.StatusInternalServerError, "failed to get storage usage")
			return
		}

		var totalSize int64
		for _, file := range files {
			totalSize += file.Size
		}

		c.JSON(http.StatusOK, gin.H{
			"storage_id":   storageID,
			"access_scope": accessScope,
			"total_size":   totalSize,
			"file_count":   int64(len(files)),
		})
		return
	}

	// Otherwise use the optimized GetStorageUsage method
	totalSize, fileCount, err := h.fileRepo.GetStorageUsage(c.Request.Context(), storageID)
	if err != nil {
		h.logger.Error("Failed to get storage usage", "error", err, "storage_id", storageID)
		respondError(c, http.StatusInternalServerError, "failed to get storage usage")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"storage_id": storageID,
		"total_size": totalSize,
		"file_count": fileCount,
	})
}

// Helper methods

func (h *FileHandlers) entryToModel(entry *models.FileEntry) *storagemodels.FileModel {
	model := &storagemodels.FileModel{
		StorageID:   entry.StorageID,
		Name:        entry.Name,
		Path:        entry.Path,
		MimeType:    entry.MimeType,
		Size:        entry.Size,
		Checksum:    entry.Checksum,
		AccessScope: string(entry.AccessScope),
		Tags:        storagemodels.StringArray(entry.Tags),
		Metadata:    storagemodels.JSONBMap(entry.Metadata),
	}

	// Parse ID
	if id, err := uuid.Parse(entry.ID); err == nil {
		model.ID = id
	}

	// Set optional fields
	if entry.WorkflowID != nil {
		if wfUUID, err := uuid.Parse(*entry.WorkflowID); err == nil {
			model.WorkflowID = &wfUUID
		}
	}
	if entry.ExecutionID != nil {
		if exUUID, err := uuid.Parse(*entry.ExecutionID); err == nil {
			model.ExecutionID = &exUUID
		}
	}
	if entry.SourceNodeID != nil {
		model.SourceNodeID = entry.SourceNodeID
	}
	if entry.TTL != nil {
		ttlSeconds := int(entry.TTL.Seconds())
		model.TTLSeconds = &ttlSeconds
	}
	if entry.ExpiresAt != nil {
		model.ExpiresAt = entry.ExpiresAt
	}

	return model
}

func (h *FileHandlers) modelToResponse(model *storagemodels.FileModel) map[string]any {
	resp := map[string]any{
		"id":           model.ID.String(),
		"storage_id":   model.StorageID,
		"name":         model.Name,
		"path":         model.Path,
		"mime_type":    model.MimeType,
		"size":         model.Size,
		"checksum":     model.Checksum,
		"access_scope": model.AccessScope,
		"tags":         []string(model.Tags),
		"metadata":     model.Metadata,
		"created_at":   model.CreatedAt,
		"updated_at":   model.UpdatedAt,
	}

	if model.WorkflowID != nil {
		resp["workflow_id"] = model.WorkflowID.String()
	}
	if model.ExecutionID != nil {
		resp["execution_id"] = model.ExecutionID.String()
	}
	if model.SourceNodeID != nil {
		resp["source_node_id"] = *model.SourceNodeID
	}
	if model.ExpiresAt != nil {
		resp["expires_at"] = model.ExpiresAt
	}

	return resp
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
