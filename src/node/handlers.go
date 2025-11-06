package node

import (
	"github.com/gin-gonic/gin"
	"mbflow/internal/db"
	"net/http"
)

// CreateTemplateRequest - структура запроса для создания темплейта
type CreateTemplateRequest struct {
	Type        string         `json:"type" binding:"required"`
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// CreateTemplateHandler - обработчик создания темплейта
func CreateTemplateHandler(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template := Template{
		Type:        Type(req.Type),
		Name:        req.Name,
		Description: req.Description,
		Parameters:  req.Parameters,
	}

	dbConn := db.DB()
	_, err := dbConn.NewInsert().Model(&template).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// GetTemplateHandler - получение темплейта по ID
func GetTemplateHandler(c *gin.Context) {
	templateID := c.Param("id")
	var template Template

	dbConn := db.DB()
	err := dbConn.NewSelect().Model(&template).Where("id = ?", templateID).Scan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// ListTemplatesHandler - получение списка всех темплейтов
func ListTemplatesHandler(c *gin.Context) {
	var templates []Template
	dbConn := db.DB()
	err := dbConn.NewSelect().Model(&templates).Scan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch templates"})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// UpdateTemplateHandler - обновление темплейта
func UpdateTemplateHandler(c *gin.Context) {
	templateID := c.Param("id")
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbConn := db.DB()
	_, err := dbConn.NewUpdate().Model(&Template{}).
		Set("type = ?", req.Type).
		Set("name = ?", req.Name).
		Set("description = ?", req.Description).
		Set("parameters = ?", req.Parameters).
		Where("id = ?", templateID).
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template updated successfully"})
}

// DeleteTemplateHandler - удаление темплейта
func DeleteTemplateHandler(c *gin.Context) {
	templateID := c.Param("id")
	dbConn := db.DB()
	_, err := dbConn.NewDelete().Model(&Template{}).
		Where("id = ?", templateID).
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

// RegisterTemplateRoutes - регистрация маршрутов для работы с темплейтами
func RegisterTemplateRoutes(router *gin.Engine) {
	templates := router.Group("/templates")
	templates.POST("", CreateTemplateHandler)
	templates.GET("/:id", GetTemplateHandler)
	templates.GET("", ListTemplatesHandler)
	templates.PUT("/:id", UpdateTemplateHandler)
	templates.DELETE("/:id", DeleteTemplateHandler)
}
