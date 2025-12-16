package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
)

type FileStorageHandlers struct {
	resourceRepo repository.FileStorageRepository
	fileService  *filestorage.ResourceFileService
	logger       *logger.Logger
}

func NewFileStorageHandlers(
	resourceRepo repository.FileStorageRepository,
	fileService *filestorage.ResourceFileService,
	log *logger.Logger,
) *FileStorageHandlers {
	return &FileStorageHandlers{
		resourceRepo: resourceRepo,
		fileService:  fileService,
		logger:       log,
	}
}

func (h *FileStorageHandlers) UploadFile(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resourceID, ok := getParam(c, "id")
	if !ok {
		return
	}

	resource, err := h.resourceRepo.GetByID(c.Request.Context(), resourceID)
	if err != nil {
		if err == models.ErrResourceNotFound {
			respondError(c, http.StatusNotFound, "resource not found")
			return
		}
		h.logger.Error("Failed to get resource", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to get resource")
		return
	}

	if resource.GetOwnerID() != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])
	file.Seek(0, 0)

	if !models.IsMimeTypeAllowed(mimeType) {
		respondError(c, http.StatusBadRequest, fmt.Sprintf("MIME type not allowed: %s", mimeType))
		return
	}

	fileModel, err := h.fileService.UploadFile(
		c.Request.Context(),
		resourceID,
		header.Filename,
		header.Size,
		mimeType,
		file,
	)
	if err != nil {
		h.logger.Error("Failed to upload file", "error", err, "resource_id", resourceID, "filename", header.Filename)
		if strings.Contains(err.Error(), "quota exceeded") {
			respondError(c, http.StatusInsufficientStorage, err.Error())
			return
		}
		if strings.Contains(err.Error(), "exceeds maximum") {
			respondError(c, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to upload file")
		return
	}

	h.logger.Info("File uploaded",
		"resource_id", resourceID,
		"file_id", fileModel.ID,
		"filename", header.Filename,
		"size", header.Size,
	)

	respondJSON(c, http.StatusCreated, gin.H{
		"id":         fileModel.ID.String(),
		"name":       fileModel.Name,
		"size":       fileModel.Size,
		"mime_type":  fileModel.MimeType,
		"checksum":   fileModel.Checksum,
		"created_at": fileModel.CreatedAt,
	})
}

func (h *FileStorageHandlers) ListFiles(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resourceID, ok := getParam(c, "id")
	if !ok {
		return
	}

	resource, err := h.resourceRepo.GetByID(c.Request.Context(), resourceID)
	if err != nil {
		if err == models.ErrResourceNotFound {
			respondError(c, http.StatusNotFound, "resource not found")
			return
		}
		h.logger.Error("Failed to get resource", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to get resource")
		return
	}

	if resource.GetOwnerID() != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	files, total, err := h.fileService.ListFiles(c.Request.Context(), resourceID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list files", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to list files")
		return
	}

	response := make([]gin.H, len(files))
	for i, f := range files {
		response[i] = gin.H{
			"id":         f.ID.String(),
			"name":       f.Name,
			"size":       f.Size,
			"mime_type":  f.MimeType,
			"checksum":   f.Checksum,
			"created_at": f.CreatedAt,
			"updated_at": f.UpdatedAt,
		}
		if f.ExpiresAt != nil {
			response[i]["expires_at"] = f.ExpiresAt
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"files":  response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *FileStorageHandlers) GetFileMetadata(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resourceID, ok := getParam(c, "id")
	if !ok {
		return
	}

	fileID, ok := getParam(c, "file_id")
	if !ok {
		return
	}

	resource, err := h.resourceRepo.GetByID(c.Request.Context(), resourceID)
	if err != nil {
		if err == models.ErrResourceNotFound {
			respondError(c, http.StatusNotFound, "resource not found")
			return
		}
		h.logger.Error("Failed to get resource", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to get resource")
		return
	}

	if resource.GetOwnerID() != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	fileModel, err := h.fileService.GetFileMetadata(c.Request.Context(), resourceID, fileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(c, http.StatusNotFound, "file not found")
			return
		}
		if strings.Contains(err.Error(), "does not belong") {
			respondError(c, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to get file metadata", "error", err, "file_id", fileID)
		respondError(c, http.StatusInternalServerError, "failed to get file metadata")
		return
	}

	resp := gin.H{
		"id":         fileModel.ID.String(),
		"name":       fileModel.Name,
		"size":       fileModel.Size,
		"mime_type":  fileModel.MimeType,
		"checksum":   fileModel.Checksum,
		"path":       fileModel.Path,
		"created_at": fileModel.CreatedAt,
		"updated_at": fileModel.UpdatedAt,
	}

	if fileModel.ExpiresAt != nil {
		resp["expires_at"] = fileModel.ExpiresAt
	}

	respondJSON(c, http.StatusOK, resp)
}

func (h *FileStorageHandlers) DownloadFile(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resourceID, ok := getParam(c, "id")
	if !ok {
		return
	}

	fileID, ok := getParam(c, "file_id")
	if !ok {
		return
	}

	resource, err := h.resourceRepo.GetByID(c.Request.Context(), resourceID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "resource not found")
			return
		}
		h.logger.Error("Failed to get resource", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to get resource")
		return
	}

	if resource.GetOwnerID() != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	fileModel, reader, err := h.fileService.GetFile(c.Request.Context(), resourceID, fileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(c, http.StatusNotFound, "file not found")
			return
		}
		if strings.Contains(err.Error(), "expired") {
			respondError(c, http.StatusGone, "file has expired")
			return
		}
		if strings.Contains(err.Error(), "does not belong") {
			respondError(c, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to get file", "error", err, "file_id", fileID)
		respondError(c, http.StatusInternalServerError, "failed to retrieve file")
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileModel.Name))
	c.Header("Content-Type", fileModel.MimeType)

	c.DataFromReader(http.StatusOK, fileModel.Size, fileModel.MimeType, reader, nil)
}

func (h *FileStorageHandlers) DeleteFile(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resourceID, ok := getParam(c, "id")
	if !ok {
		return
	}

	fileID, ok := getParam(c, "file_id")
	if !ok {
		return
	}

	resource, err := h.resourceRepo.GetByID(c.Request.Context(), resourceID)
	if err != nil {
		if err == models.ErrResourceNotFound {
			respondError(c, http.StatusNotFound, "resource not found")
			return
		}
		h.logger.Error("Failed to get resource", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to get resource")
		return
	}

	if resource.GetOwnerID() != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	if err := h.fileService.DeleteFile(c.Request.Context(), resourceID, fileID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(c, http.StatusNotFound, "file not found")
			return
		}
		if strings.Contains(err.Error(), "does not belong") {
			respondError(c, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to delete file", "error", err, "file_id", fileID)
		respondError(c, http.StatusInternalServerError, "failed to delete file")
		return
	}

	h.logger.Info("File deleted", "resource_id", resourceID, "file_id", fileID, "user_id", userID)

	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}
