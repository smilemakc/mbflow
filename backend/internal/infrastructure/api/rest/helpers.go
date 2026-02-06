package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

// respondList writes a paginated list response with flat structure
func respondList(c *gin.Context, status int, data interface{}, total, limit, offset int) {
	c.JSON(status, gin.H{
		"data":   data,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
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

// respondSuccess writes a successful response, optionally with pagination metadata
func respondSuccess(c *gin.Context, status int, data interface{}, meta *listMeta) {
	if meta != nil {
		c.JSON(status, gin.H{
			"data":   data,
			"total":  meta.Total,
			"limit":  meta.Limit,
			"offset": meta.Offset,
		})
	} else {
		c.JSON(status, data)
	}
}

// listMeta contains pagination metadata for list responses
type listMeta struct {
	Total  int
	Limit  int
	Offset int
}

func bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		var ve validator.ValidationErrors
		if ok := errors.As(err, &ve); ok {
			msgs := make([]string, 0, len(ve))
			for _, fe := range ve {
				field := strings.ToLower(fe.Field())
				switch fe.Tag() {
				case "required":
					msgs = append(msgs, fmt.Sprintf("%s is required", field))
				case "uuid":
					msgs = append(msgs, fmt.Sprintf("%s must be a valid UUID", field))
				case "min":
					msgs = append(msgs, fmt.Sprintf("%s must be at least %s characters", field, fe.Param()))
				case "max":
					msgs = append(msgs, fmt.Sprintf("%s must be at most %s characters", field, fe.Param()))
				default:
					msgs = append(msgs, fmt.Sprintf("%s is invalid", field))
				}
			}
			respondError(c, http.StatusBadRequest, strings.Join(msgs, "; "))
		} else {
			respondAPIError(c, ErrInvalidJSON)
		}
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
