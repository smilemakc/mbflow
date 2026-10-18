package rest

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

// Ownership model
//
//   - workflows:  mbflow_workflows.created_by
//   - nodes/edges/resources of a workflow, triggers, stored executions:
//     owned through their workflow
//   - inline (ephemeral) executions: mbflow_executions.created_by
//   - files: mbflow_files.created_by, or the workflow they are attached to
//
// Administrators can access everything. Records without an owner (legacy
// rows created before ownership was tracked) are visible to admins only.
// Records owned by someone else are reported as 404, never 403, so their
// existence is not disclosed.

// accessScope describes which records the current caller may access.
type accessScope struct {
	userID uuid.UUID
	admin  bool
}

// scopeFromContext returns the caller's access scope. ok is false when the
// request carries no authenticated user.
func scopeFromContext(c *gin.Context) (accessScope, bool) {
	if IsAdmin(c) {
		userID, _ := GetUserIDAsUUID(c)
		return accessScope{userID: userID, admin: true}, true
	}
	userID, ok := GetUserIDAsUUID(c)
	if !ok || userID == uuid.Nil {
		return accessScope{}, false
	}
	return accessScope{userID: userID}, true
}

// owns reports whether the caller may access a record owned by owner.
func (s accessScope) owns(owner *uuid.UUID) bool {
	return s.admin || (owner != nil && *owner == s.userID)
}

// ownerFilterFromContext returns the user ID that listings must be filtered
// by, or nil for administrators. Without an authenticated caller (handlers
// used directly, e.g. in tests) no filter is applied; production routes are
// always behind RequireAuth.
func ownerFilterFromContext(c *gin.Context) *uuid.UUID {
	scope, ok := scopeFromContext(c)
	if !ok || scope.admin {
		return nil
	}
	id := scope.userID
	return &id
}

// isNotFound normalizes the different "not found" conventions of the
// repositories (sql.ErrNoRows, domain errors, "... not found" messages).
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) ||
		errors.Is(err, pkgmodels.ErrWorkflowNotFound) ||
		errors.Is(err, pkgmodels.ErrExecutionNotFound) ||
		errors.Is(err, pkgmodels.ErrTriggerNotFound) ||
		errors.Is(err, pkgmodels.ErrResourceNotFound) {
		return true
	}
	return strings.Contains(err.Error(), "not found")
}

// OwnershipGuard enforces per-user isolation for workflow-related resources.
type OwnershipGuard struct {
	workflows  repository.WorkflowRepository
	executions repository.ExecutionRepository
	triggers   repository.TriggerRepository
	files      *storage.FileRepository
	resources  repository.ResourceRepository
	logger     *logger.Logger
}

// NewOwnershipGuard creates an ownership guard. Repositories that are nil
// disable the corresponding checks' lookups (the record is treated as absent).
func NewOwnershipGuard(
	workflows repository.WorkflowRepository,
	executions repository.ExecutionRepository,
	triggers repository.TriggerRepository,
	files *storage.FileRepository,
	resources repository.ResourceRepository,
	log *logger.Logger,
) *OwnershipGuard {
	return &OwnershipGuard{
		workflows:  workflows,
		executions: executions,
		triggers:   triggers,
		files:      files,
		resources:  resources,
		logger:     log,
	}
}

// canAccessWorkflow reports whether the caller may access the workflow.
// A missing workflow yields (false, nil).
func (g *OwnershipGuard) canAccessWorkflow(ctx context.Context, scope accessScope, id uuid.UUID) (bool, error) {
	if g.workflows == nil {
		return false, nil
	}
	wf, err := g.workflows.FindByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if wf == nil {
		return false, nil
	}
	return scope.owns(wf.CreatedBy), nil
}

func (g *OwnershipGuard) canAccessExecution(ctx context.Context, scope accessScope, id uuid.UUID) (bool, error) {
	if g.executions == nil {
		return false, nil
	}
	exec, err := g.executions.FindByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if exec == nil {
		return false, nil
	}
	if scope.admin {
		return true, nil
	}
	if exec.CreatedBy != nil && *exec.CreatedBy == scope.userID {
		return true, nil
	}
	if exec.WorkflowID != nil {
		return g.canAccessWorkflow(ctx, scope, *exec.WorkflowID)
	}
	return false, nil
}

func (g *OwnershipGuard) canAccessTrigger(ctx context.Context, scope accessScope, id uuid.UUID) (bool, error) {
	if g.triggers == nil {
		return false, nil
	}
	trigger, err := g.triggers.FindByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if trigger == nil {
		return false, nil
	}
	if scope.admin {
		return true, nil
	}
	return g.canAccessWorkflow(ctx, scope, trigger.WorkflowID)
}

func (g *OwnershipGuard) canAccessFile(ctx context.Context, scope accessScope, id uuid.UUID) (bool, error) {
	if g.files == nil {
		return false, nil
	}
	file, err := g.files.FindByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if file == nil {
		return false, nil
	}
	if scope.owns(file.CreatedBy) {
		return true, nil
	}
	if file.WorkflowID != nil {
		return g.canAccessWorkflow(ctx, scope, *file.WorkflowID)
	}
	return false, nil
}

func (g *OwnershipGuard) canAccessResource(ctx context.Context, scope accessScope, id uuid.UUID) (bool, error) {
	if scope.admin {
		return true, nil
	}
	if g.resources == nil {
		return false, nil
	}
	res, err := g.resources.GetByID(ctx, id.String())
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return res != nil && res.GetOwnerID() == scope.userID.String(), nil
}

// CheckWorkflowAccess verifies the caller may access workflowID. On failure it
// writes the response (401/404/500) and returns false.
func (g *OwnershipGuard) CheckWorkflowAccess(c *gin.Context, workflowID uuid.UUID) bool {
	return g.check(c, "WORKFLOW_NOT_FOUND", "Workflow not found", func(ctx context.Context, s accessScope) (bool, error) {
		return g.canAccessWorkflow(ctx, s, workflowID)
	})
}

// CheckResourceAccess verifies the caller owns resourceID (admins bypass).
func (g *OwnershipGuard) CheckResourceAccess(c *gin.Context, resourceID uuid.UUID) bool {
	return g.check(c, "RESOURCE_NOT_FOUND", "Resource not found", func(ctx context.Context, s accessScope) (bool, error) {
		return g.canAccessResource(ctx, s, resourceID)
	})
}

func (g *OwnershipGuard) check(c *gin.Context, code, message string, allowed func(context.Context, accessScope) (bool, error)) bool {
	scope, ok := scopeFromContext(c)
	if !ok {
		respondAPIError(c, ErrUnauthorized)
		return false
	}
	ok, err := allowed(c.Request.Context(), scope)
	if err != nil {
		if g.logger != nil {
			g.logger.Error("Ownership check failed", "error", err, "request_id", GetRequestID(c))
		}
		respondAPIError(c, NewAPIError("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError))
		return false
	}
	if !ok {
		respondAPIError(c, NewAPIError(code, message, http.StatusNotFound))
		return false
	}
	return true
}

// middleware builds a handler that checks the ID found by extract. Requests
// without that ID, or with a malformed one, are passed through so the route
// handler can report its usual validation error.
func (g *OwnershipGuard) middleware(extract func(*gin.Context) string, code, message string, allowed func(context.Context, accessScope, uuid.UUID) (bool, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extract(c)
		if raw == "" {
			c.Next()
			return
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			c.Next()
			return
		}
		if !g.check(c, code, message, func(ctx context.Context, s accessScope) (bool, error) {
			return allowed(ctx, s, id)
		}) {
			c.Abort()
			return
		}
		c.Next()
	}
}

func paramExtractor(name string) func(*gin.Context) string {
	return func(c *gin.Context) string { return c.Param(name) }
}

// RequireWorkflowAccess guards routes with a workflow ID path parameter.
func (g *OwnershipGuard) RequireWorkflowAccess(param string) gin.HandlerFunc {
	return g.middleware(paramExtractor(param), "WORKFLOW_NOT_FOUND", "Workflow not found", g.canAccessWorkflow)
}

// RequireExecutionAccess guards routes with an execution ID path parameter.
func (g *OwnershipGuard) RequireExecutionAccess(param string) gin.HandlerFunc {
	return g.middleware(paramExtractor(param), "EXECUTION_NOT_FOUND", "Execution not found", g.canAccessExecution)
}

// RequireExecutionQueryAccess guards routes with an execution ID query parameter.
func (g *OwnershipGuard) RequireExecutionQueryAccess(query string) gin.HandlerFunc {
	return g.middleware(func(c *gin.Context) string { return c.Query(query) },
		"EXECUTION_NOT_FOUND", "Execution not found", g.canAccessExecution)
}

// RequireTriggerAccess guards routes with a trigger ID path parameter.
func (g *OwnershipGuard) RequireTriggerAccess(param string) gin.HandlerFunc {
	return g.middleware(paramExtractor(param), "TRIGGER_NOT_FOUND", "Trigger not found", g.canAccessTrigger)
}

// RequireFileAccess guards routes with a file ID path parameter.
func (g *OwnershipGuard) RequireFileAccess(param string) gin.HandlerFunc {
	return g.middleware(paramExtractor(param), "FILE_NOT_FOUND", "File not found", g.canAccessFile)
}
