package rest

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ResourceHandlers handles resource-related HTTP requests
type ResourceHandlers struct {
	resourceRepo repository.FileStorageRepository
	planRepo     repository.PricingPlanRepository
	workflowRepo repository.WorkflowRepository
	logger       *logger.Logger
}

// NewResourceHandlers creates a new ResourceHandlers instance
func NewResourceHandlers(resourceRepo repository.FileStorageRepository, planRepo repository.PricingPlanRepository, workflowRepo repository.WorkflowRepository, log *logger.Logger) *ResourceHandlers {
	return &ResourceHandlers{
		resourceRepo: resourceRepo,
		planRepo:     planRepo,
		workflowRepo: workflowRepo,
		logger:       log,
	}
}

// CreateFileStorageRequest represents request to create file storage resource
type CreateFileStorageRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
}

// CreateFileStorageResponse represents response after creating file storage
type CreateFileStorageResponse struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Status            string    `json:"status"`
	StorageLimitBytes int64     `json:"storage_limit_bytes"`
	UsedStorageBytes  int64     `json:"used_storage_bytes"`
	FileCount         int       `json:"file_count"`
	UsagePercent      float64   `json:"usage_percent"`
	CreatedAt         time.Time `json:"created_at"`
}

// CreateFileStorage creates a new file storage resource
// POST /api/v1/resources/file-storage
func (h *ResourceHandlers) CreateFileStorage(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateFileStorageRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	fsResource := models.NewFileStorageResource(userID, req.Name)
	fsResource.Description = req.Description

	if err := h.resourceRepo.Create(c.Request.Context(), fsResource); err != nil {
		h.logger.Error("Failed to create file storage resource", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to create resource")
		return
	}

	h.logger.Info("File storage resource created",
		"resource_id", fsResource.ID,
		"user_id", userID,
		"name", fsResource.Name,
	)

	respondJSON(c, http.StatusCreated, CreateFileStorageResponse{
		ID:                fsResource.ID,
		Name:              fsResource.Name,
		Description:       fsResource.Description,
		Status:            string(fsResource.Status),
		StorageLimitBytes: fsResource.StorageLimitBytes,
		UsedStorageBytes:  fsResource.UsedStorageBytes,
		FileCount:         fsResource.FileCount,
		UsagePercent:      fsResource.GetUsagePercent(),
		CreatedAt:         fsResource.CreatedAt,
	})
}

// ListResources returns all resources for the current user
// GET /api/v1/resources
func (h *ResourceHandlers) ListResources(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resources, err := h.resourceRepo.GetByOwner(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get resources", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to get resources")
		return
	}

	response := make([]gin.H, 0, len(resources))
	for _, r := range resources {
		if r == nil {
			continue
		}
		response = append(response, h.resourceToResponse(r))
	}

	c.JSON(http.StatusOK, gin.H{"resources": response})
}

// resourceToResponse converts a Resource to a gin.H response with type-specific fields
func (h *ResourceHandlers) resourceToResponse(r models.Resource) gin.H {
	resp := gin.H{
		"id":          r.GetID(),
		"type":        r.GetType(),
		"name":        r.GetName(),
		"description": r.GetDescription(),
		"status":      r.GetStatus(),
	}

	// Add type-specific fields
	switch res := r.(type) {
	case *models.FileStorageResource:
		resp["storage_limit_bytes"] = res.StorageLimitBytes
		resp["used_storage_bytes"] = res.UsedStorageBytes
		resp["file_count"] = res.FileCount
		resp["usage_percent"] = res.GetUsagePercent()
		resp["created_at"] = res.CreatedAt
		resp["updated_at"] = res.UpdatedAt
	case *models.CredentialsResource:
		resp["credential_type"] = res.CredentialType
		resp["provider"] = res.Provider
		resp["expires_at"] = res.ExpiresAt
		resp["last_used_at"] = res.LastUsedAt
		resp["usage_count"] = res.UsageCount
		resp["created_at"] = res.CreatedAt
		resp["updated_at"] = res.UpdatedAt
	case *models.RentalKeyResource:
		resp["provider"] = res.Provider
		resp["daily_request_limit"] = res.DailyRequestLimit
		resp["monthly_token_limit"] = res.MonthlyTokenLimit
		resp["requests_today"] = res.RequestsToday
		resp["tokens_this_month"] = res.TokensThisMonth
		resp["total_requests"] = res.TotalRequests
		resp["total_cost"] = res.TotalCost
		resp["last_used_at"] = res.LastUsedAt
		resp["created_at"] = res.CreatedAt
		resp["updated_at"] = res.UpdatedAt
	}

	return resp
}

// GetResource returns a specific resource by ID
// GET /api/v1/resources/:id
func (h *ResourceHandlers) GetResource(c *gin.Context) {
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

	resp := h.resourceToResponse(resource)
	// Add available_space for FileStorage
	if fs, ok := resource.(*models.FileStorageResource); ok {
		resp["available_space"] = fs.GetAvailableSpace()
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateResourceRequest represents request to update resource
type UpdateResourceRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
}

// UpdateResource updates a resource
// PUT /api/v1/resources/:id
func (h *ResourceHandlers) UpdateResource(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resourceID, ok := getParam(c, "id")
	if !ok {
		return
	}

	var req UpdateResourceRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	resource, err := h.resourceRepo.GetByID(c.Request.Context(), resourceID)
	if err != nil {
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "resource not found")
			return
		}
		h.logger.Error("Failed to get resource for update", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to get resource")
		return
	}

	if resource.GetOwnerID() != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	// Update base fields using type switch
	switch res := resource.(type) {
	case *models.FileStorageResource:
		if req.Name != "" {
			res.Name = req.Name
		}
		res.Description = req.Description
		res.UpdatedAt = time.Now()
	case *models.CredentialsResource:
		if req.Name != "" {
			res.Name = req.Name
		}
		res.Description = req.Description
		res.UpdatedAt = time.Now()
	case *models.RentalKeyResource:
		// Rental keys should be updated via admin endpoints
		respondError(c, http.StatusForbidden, "rental keys must be updated via admin endpoints")
		return
	default:
		respondError(c, http.StatusInternalServerError, "unsupported resource type")
		return
	}

	if err := h.resourceRepo.Update(c.Request.Context(), resource); err != nil {
		h.logger.Error("Failed to update resource", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to update resource")
		return
	}

	h.logger.Info("Resource updated", "resource_id", resourceID, "user_id", userID)

	resp := h.resourceToResponse(resource)
	if fs, ok := resource.(*models.FileStorageResource); ok {
		resp["available_space"] = fs.GetAvailableSpace()
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteResource soft-deletes a resource
// DELETE /api/v1/resources/:id
func (h *ResourceHandlers) DeleteResource(c *gin.Context) {
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
		if errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "resource not found")
			return
		}
		h.logger.Error("Failed to get resource for deletion", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to get resource")
		return
	}

	if resource.GetOwnerID() != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	// First, detach resource from all workflows
	resourceUUID, err := uuid.Parse(resourceID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid resource ID")
		return
	}

	detachedCount, err := h.workflowRepo.UnassignResourceFromAllWorkflows(c.Request.Context(), resourceUUID)
	if err != nil {
		h.logger.Error("Failed to detach resource from workflows", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to detach resource from workflows")
		return
	}

	if detachedCount > 0 {
		h.logger.Info("Resource detached from workflows", "resource_id", resourceID, "workflows_count", detachedCount)
	}

	// Then delete the resource
	if err := h.resourceRepo.Delete(c.Request.Context(), resourceID); err != nil {
		h.logger.Error("Failed to delete resource", "error", err, "resource_id", resourceID)
		respondError(c, http.StatusInternalServerError, "failed to delete resource")
		return
	}

	h.logger.Info("Resource deleted", "resource_id", resourceID, "user_id", userID, "detached_from_workflows", detachedCount)

	c.JSON(http.StatusOK, gin.H{"message": "resource deleted successfully"})
}

// ListPricingPlans returns available pricing plans for file storage
// GET /api/v1/resources/pricing-plans
func (h *ResourceHandlers) ListPricingPlans(c *gin.Context) {
	resourceType := c.DefaultQuery("resource_type", string(models.ResourceTypeFileStorage))

	plans, err := h.planRepo.GetByResourceType(c.Request.Context(), models.ResourceType(resourceType))
	if err != nil {
		h.logger.Error("Failed to get pricing plans", "error", err, "resource_type", resourceType)
		respondError(c, http.StatusInternalServerError, "failed to get pricing plans")
		return
	}

	response := make([]gin.H, len(plans))
	for i, plan := range plans {
		response[i] = gin.H{
			"id":                  plan.ID,
			"resource_type":       plan.ResourceType,
			"name":                plan.Name,
			"description":         plan.Description,
			"price_per_unit":      plan.PricePerUnit,
			"unit":                plan.Unit,
			"storage_limit_bytes": plan.StorageLimitBytes,
			"billing_period":      plan.BillingPeriod,
			"pricing_model":       plan.PricingModel,
			"is_free":             plan.IsFree,
			"is_active":           plan.IsActive,
			"monthly_price":       plan.GetMonthlyPrice(),
			"annual_price":        plan.GetAnnualPrice(),
		}
	}

	c.JSON(http.StatusOK, gin.H{"plans": response})
}
