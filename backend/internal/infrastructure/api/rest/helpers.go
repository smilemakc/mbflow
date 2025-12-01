package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseIntQuery parses integer query parameter with default value
func parseIntQuery(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}

// respondJSON writes JSON response using Gin
func respondJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// respondError writes error response using Gin
func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// respondErrorWithDetails writes a detailed error response
func respondErrorWithDetails(c *gin.Context, status int, message, code string, details map[string]interface{}) {
	c.JSON(status, ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}

// SuccessResponse represents a successful response with metadata
type SuccessResponse struct {
	Data interface{} `json:"data"`
	Meta *MetaInfo   `json:"meta,omitempty"`
}

// MetaInfo contains metadata about the response
type MetaInfo struct {
	Total  int `json:"total,omitempty"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// respondSuccess writes a successful response with metadata
func respondSuccess(c *gin.Context, status int, data interface{}, meta *MetaInfo) {
	if meta != nil {
		c.JSON(status, SuccessResponse{
			Data: data,
			Meta: meta,
		})
	} else {
		c.JSON(status, gin.H{"data": data})
	}
}

// bindJSON is a helper to bind JSON request body
func bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return err
	}
	return nil
}

// getParam gets a path parameter and validates it's not empty
func getParam(c *gin.Context, name string) (string, bool) {
	value := c.Param(name)
	if value == "" {
		respondError(c, http.StatusBadRequest, name+" is required")
		return "", false
	}
	return value, true
}

// getQuery gets a query parameter with a default value
func getQuery(c *gin.Context, name string, defaultValue string) string {
	value := c.Query(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// getQueryInt gets a query parameter as integer with a default value
func getQueryInt(c *gin.Context, name string, defaultValue int) int {
	value := c.Query(name)
	return parseIntQuery(value, defaultValue)
}
