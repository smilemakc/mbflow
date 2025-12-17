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

func respondJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

func respondError(c *gin.Context, status int, message string) {
	apiErr := NewAPIError("ERROR", message, status)
	c.JSON(status, apiErr)
}

func respondErrorWithDetails(c *gin.Context, status int, message, code string, details map[string]interface{}) {
	apiErr := NewAPIErrorWithDetails(code, message, status, details)
	c.JSON(status, apiErr)
}

func respondAPIError(c *gin.Context, err error) {
	apiErr := TranslateError(err)
	c.JSON(apiErr.HTTPStatus, apiErr)
}

func respondAPIErrorWithRequestID(c *gin.Context, err error) {
	apiErr := TranslateError(err)
	if apiErr.Details == nil {
		apiErr.Details = make(map[string]interface{})
	}
	apiErr.Details["request_id"] = GetRequestID(c)
	c.JSON(apiErr.HTTPStatus, apiErr)
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

func bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		respondAPIError(c, ErrInvalidJSON)
		return err
	}
	return nil
}

func getParam(c *gin.Context, name string) (string, bool) {
	value := c.Param(name)
	if value == "" {
		respondAPIErrorWithRequestID(c, NewAPIError("MISSING_PARAMETER", name+" is required", http.StatusBadRequest))
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
