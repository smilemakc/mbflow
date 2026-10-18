package rest

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/smilemakc/mbflow/go/internal/application/apiclient"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// APIClientHandlers serves API clients imported from OpenAPI specifications.
type APIClientHandlers struct {
	service *apiclient.Service
	logger  *logger.Logger
}

// NewAPIClientHandlers creates APIClientHandlers.
func NewAPIClientHandlers(service *apiclient.Service, log *logger.Logger) *APIClientHandlers {
	return &APIClientHandlers{service: service, logger: log}
}

// specUpload is the payload of import, preview and specification update requests.
// A multipart form carries the document in the "file" field and the settings
// as fields ("auth" and "default_headers" as JSON); a JSON body carries the
// document as text in "spec"; any other body is the document itself.
type specUpload struct {
	Spec           string                `json:"spec"`
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	BaseURL        string                `json:"base_url"`
	Auth           *models.APIClientAuth `json:"auth"`
	DefaultHeaders map[string]string     `json:"default_headers"`
}

type updateAPIClientRequest struct {
	Name           *string               `json:"name"`
	Description    *string               `json:"description"`
	BaseURL        *string               `json:"base_url"`
	Auth           *models.APIClientAuth `json:"auth"`
	DefaultHeaders map[string]string     `json:"default_headers"`
}

type apiClientSummary struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description,omitempty"`
	Title          string                  `json:"title"`
	APIVersion     string                  `json:"api_version,omitempty"`
	SpecVersion    string                  `json:"spec_version"`
	BaseURL        string                  `json:"base_url,omitempty"`
	Auth           models.APIClientAuth    `json:"auth"`
	OperationCount int                     `json:"operation_count"`
	Warnings       []models.APISpecWarning `json:"warnings"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

type operationSummary struct {
	ID         string   `json:"id"`
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	Summary    string   `json:"summary,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Deprecated bool     `json:"deprecated,omitempty"`
}

type specPreviewResponse struct {
	Title           string                      `json:"title"`
	Description     string                      `json:"description,omitempty"`
	APIVersion      string                      `json:"api_version,omitempty"`
	SpecVersion     string                      `json:"spec_version"`
	BaseURL         string                      `json:"base_url,omitempty"`
	Servers         []models.APIServer          `json:"servers"`
	SecuritySchemes []*models.APISecurityScheme `json:"security_schemes"`
	SuggestedAuth   models.APIClientAuth        `json:"suggested_auth"`
	OperationCount  int                         `json:"operation_count"`
	Operations      []operationSummary          `json:"operations"`
	Warnings        []models.APISpecWarning     `json:"warnings"`
}

type replaceSpecResponse struct {
	Client  *models.APIClient      `json:"client"`
	Changes *apiclient.SpecChanges `json:"changes"`
}

// HandlePreview parses a specification without saving it.
// POST /api/v1/api-clients/preview
func (h *APIClientHandlers) HandlePreview(c *gin.Context) {
	upload, ok := h.readSpecUpload(c)
	if !ok {
		return
	}
	spec, err := h.service.Preview(c.Request.Context(), []byte(upload.Spec))
	if err != nil {
		h.respondError(c, err)
		return
	}

	ops := make([]operationSummary, 0, len(spec.Operations))
	for _, op := range spec.Operations {
		ops = append(ops, summarizeOperation(op))
	}
	respondJSON(c, http.StatusOK, specPreviewResponse{
		Title:           spec.Title,
		Description:     spec.Description,
		APIVersion:      spec.APIVersion,
		SpecVersion:     spec.SpecVersion,
		BaseURL:         spec.DefaultBaseURL(),
		Servers:         nonNil(spec.Servers),
		SecuritySchemes: nonNil(spec.SecuritySchemes),
		SuggestedAuth:   spec.SuggestedAuth,
		OperationCount:  len(spec.Operations),
		Operations:      ops,
		Warnings:        nonNil(spec.Warnings),
	})
}

// HandleCreate imports a specification as a new API client.
// POST /api/v1/api-clients
func (h *APIClientHandlers) HandleCreate(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	upload, ok := h.readSpecUpload(c)
	if !ok {
		return
	}
	client, err := h.service.Create(c.Request.Context(), userID, apiclient.CreateInput{
		Name:           upload.Name,
		Description:    upload.Description,
		BaseURL:        upload.BaseURL,
		Auth:           upload.Auth,
		DefaultHeaders: upload.DefaultHeaders,
		Spec:           []byte(upload.Spec),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.logger.Info("API client imported",
		"api_client_id", client.ID, "operations", client.OperationCount, "request_id", GetRequestID(c))
	respondJSON(c, http.StatusCreated, client)
}

// HandleList lists API clients of the current user.
// GET /api/v1/api-clients?limit=&offset=
func (h *APIClientHandlers) HandleList(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	limit := getQueryInt(c, "limit", apiclient.DefaultListLimit)
	if limit <= 0 {
		limit = apiclient.DefaultListLimit
	}
	if limit > apiclient.MaxListLimit {
		limit = apiclient.MaxListLimit
	}
	offset := getQueryInt(c, "offset", 0)
	if offset < 0 {
		offset = 0
	}

	clients, total, err := h.service.List(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]apiClientSummary, 0, len(clients))
	for _, client := range clients {
		items = append(items, apiClientSummary{
			ID:             client.ID,
			Name:           client.Name,
			Description:    client.Description,
			Title:          client.Title,
			APIVersion:     client.APIVersion,
			SpecVersion:    client.SpecVersion,
			BaseURL:        client.BaseURL,
			Auth:           client.Auth,
			OperationCount: client.OperationCount,
			Warnings:       nonNil(client.Warnings),
			CreatedAt:      client.CreatedAt,
			UpdatedAt:      client.UpdatedAt,
		})
	}
	respondList(c, http.StatusOK, items, int(total), limit, offset)
}

// HandleGet returns an API client with its operations and schemas.
// GET /api/v1/api-clients/:id
func (h *APIClientHandlers) HandleGet(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	client, err := h.service.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	respondJSON(c, http.StatusOK, client)
}

// HandleUpdate changes API client settings.
// PUT /api/v1/api-clients/:id
func (h *APIClientHandlers) HandleUpdate(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req updateAPIClientRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}
	client, err := h.service.Update(c.Request.Context(), userID, c.Param("id"), apiclient.UpdateInput{
		Name:           req.Name,
		Description:    req.Description,
		BaseURL:        req.BaseURL,
		Auth:           req.Auth,
		DefaultHeaders: req.DefaultHeaders,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	respondJSON(c, http.StatusOK, client)
}

// HandleReplaceSpec re-imports the specification of an API client.
// PUT /api/v1/api-clients/:id/spec
func (h *APIClientHandlers) HandleReplaceSpec(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	upload, ok := h.readSpecUpload(c)
	if !ok {
		return
	}
	client, changes, err := h.service.ReplaceSpec(c.Request.Context(), userID, c.Param("id"), []byte(upload.Spec))
	if err != nil {
		h.respondError(c, err)
		return
	}
	respondJSON(c, http.StatusOK, replaceSpecResponse{Client: client, Changes: changes})
}

// HandleDownloadSpec returns the original specification document.
// GET /api/v1/api-clients/:id/spec
func (h *APIClientHandlers) HandleDownloadSpec(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	client, spec, err := h.service.GetSpec(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	contentType, ext := "application/yaml", "yaml"
	if strings.HasPrefix(strings.TrimSpace(spec), "{") {
		contentType, ext = "application/json", "json"
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": fileSlug(client.Name) + "." + ext,
	}))
	c.Data(http.StatusOK, contentType+"; charset=utf-8", []byte(spec))
}

// HandleDelete removes an API client.
// DELETE /api/v1/api-clients/:id
func (h *APIClientHandlers) HandleDelete(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		h.respondError(c, err)
		return
	}
	respondJSON(c, http.StatusOK, gin.H{"message": "API client deleted successfully"})
}

func (h *APIClientHandlers) respondError(c *gin.Context, err error) {
	apiErr := TranslateError(err)
	if apiErr.HTTPStatus >= http.StatusInternalServerError {
		h.logger.Error("API client request failed", "error", err, "request_id", GetRequestID(c))
	}
	respondAPIErrorWithRequestID(c, apiErr)
}

// readSpecUpload reads the document and settings from the request; on failure
// it writes the error response and returns false.
func (h *APIClientHandlers) readSpecUpload(c *gin.Context) (*specUpload, bool) {
	mediaType, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
	upload := &specUpload{}

	switch mediaType {
	case "multipart/form-data":
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			if isBodyTooLarge(err) {
				respondAPIErrorWithRequestID(c, ErrPayloadTooLarge)
			} else {
				respondAPIErrorWithRequestID(c, NewAPIError("FILE_REQUIRED", "Specification file is required in the 'file' field", http.StatusBadRequest))
			}
			return nil, false
		}
		defer func() { _ = file.Close() }()
		data, err := io.ReadAll(file)
		if err != nil {
			respondAPIErrorWithRequestID(c, NewAPIError("READ_ERROR", "Failed to read specification file", http.StatusBadRequest))
			return nil, false
		}
		upload.Spec = string(data)
		upload.Name = c.PostForm("name")
		upload.Description = c.PostForm("description")
		upload.BaseURL = c.PostForm("base_url")
		if raw := c.PostForm("auth"); raw != "" {
			upload.Auth = &models.APIClientAuth{}
			if err := json.Unmarshal([]byte(raw), upload.Auth); err != nil {
				respondAPIErrorWithRequestID(c, NewAPIError("INVALID_AUTH", "Field 'auth' must be a JSON object", http.StatusBadRequest))
				return nil, false
			}
		}
		if raw := c.PostForm("default_headers"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &upload.DefaultHeaders); err != nil {
				respondAPIErrorWithRequestID(c, NewAPIError("INVALID_HEADERS", "Field 'default_headers' must be a JSON object of strings", http.StatusBadRequest))
				return nil, false
			}
		}
	case "application/json":
		if err := bindJSON(c, upload); err != nil {
			return nil, false
		}
	default:
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			if isBodyTooLarge(err) {
				respondAPIErrorWithRequestID(c, ErrPayloadTooLarge)
			} else {
				respondAPIErrorWithRequestID(c, NewAPIError("READ_ERROR", "Failed to read request body", http.StatusBadRequest))
			}
			return nil, false
		}
		upload.Spec = string(data)
	}

	if strings.TrimSpace(upload.Spec) == "" {
		respondAPIErrorWithRequestID(c, NewAPIError("EMPTY_SPEC", "Specification document is empty", http.StatusBadRequest))
		return nil, false
	}
	return upload, true
}

func requireUserID(c *gin.Context) (string, bool) {
	userID, ok := GetUserID(c)
	if !ok || userID == "" {
		respondAPIErrorWithRequestID(c, ErrUnauthorized)
		return "", false
	}
	return userID, true
}

func isBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge) || strings.Contains(err.Error(), "request body too large")
}

func summarizeOperation(op *models.APIOperation) operationSummary {
	return operationSummary{
		ID:         op.ID,
		Method:     op.Method,
		Path:       op.Path,
		Summary:    op.Summary,
		Tags:       op.Tags,
		Deprecated: op.Deprecated,
	}
}

var nonSlugChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func fileSlug(name string) string {
	slug := strings.Trim(nonSlugChars.ReplaceAllString(name, "-"), "-.")
	if slug == "" {
		return "openapi"
	}
	return slug
}

func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
