package rest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

// Lookup fakes: each returns the configured record or error.

type rc2WorkflowLookup struct {
	repository.WorkflowRepository
	wf  *storagemodels.WorkflowModel
	err error
}

func (r *rc2WorkflowLookup) FindByID(context.Context, uuid.UUID) (*storagemodels.WorkflowModel, error) {
	return r.wf, r.err
}

type rc2ExecutionLookup struct {
	repository.ExecutionRepository
	exec *storagemodels.ExecutionModel
	err  error
}

func (r *rc2ExecutionLookup) FindByID(context.Context, uuid.UUID) (*storagemodels.ExecutionModel, error) {
	return r.exec, r.err
}

type rc2TriggerLookup struct {
	repository.TriggerRepository
	tr  *storagemodels.TriggerModel
	err error
}

func (r *rc2TriggerLookup) FindByID(context.Context, uuid.UUID) (*storagemodels.TriggerModel, error) {
	return r.tr, r.err
}

type rc2ResourceLookup struct {
	repository.ResourceRepository
	res pkgmodels.Resource
	err error
}

func (r *rc2ResourceLookup) GetByID(context.Context, string) (pkgmodels.Resource, error) {
	return r.res, r.err
}

func TestRC2_IsNotFound(t *testing.T) {
	t.Parallel()
	assert.False(t, isNotFound(nil))
	for _, err := range []error{
		sql.ErrNoRows, pkgmodels.ErrWorkflowNotFound, pkgmodels.ErrExecutionNotFound,
		pkgmodels.ErrTriggerNotFound, pkgmodels.ErrResourceNotFound,
		fmt.Errorf("wrapped: %w", sql.ErrNoRows), errors.New("record not found"),
	} {
		assert.True(t, isNotFound(err), "%v", err)
	}
	assert.False(t, isNotFound(errors.New("connection refused")))
}

// rc2GuardRouter exposes every guard middleware for the given caller.
func rc2GuardRouter(g *OwnershipGuard, userID string, admin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if userID != "" {
		r.Use(testutil.MockAuthMiddleware(userID, admin))
	}
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	r.GET("/wf/:id", g.RequireWorkflowAccess("id"), ok)
	r.GET("/ex/:id", g.RequireExecutionAccess("id"), ok)
	r.GET("/tr/:id", g.RequireTriggerAccess("id"), ok)
	r.GET("/file/:id", g.RequireFileAccess("id"), ok)
	r.GET("/res/:id", func(c *gin.Context) {
		if g.CheckResourceAccess(c, uuid.MustParse(c.Param("id"))) {
			c.Status(http.StatusNoContent)
		}
	})
	return r
}

func rc2GuardCode(t *testing.T, r http.Handler, path string) int {
	t.Helper()
	return rc2Do(t, r, http.MethodGet, path, nil).Code
}

func TestRC2_OwnershipGuard_NilRepositoriesDenyAccess(t *testing.T) {
	t.Parallel()
	g := NewOwnershipGuard(nil, nil, nil, nil, nil, nil)
	user := uuid.NewString()
	r := rc2GuardRouter(g, user, false)
	id := uuid.NewString()
	for _, p := range []string{"/wf/", "/ex/", "/tr/", "/file/", "/res/"} {
		assert.Equal(t, http.StatusNotFound, rc2GuardCode(t, r, p+id), p)
	}
	// Admins bypass the resource check even without a repository.
	assert.Equal(t, http.StatusNoContent, rc2GuardCode(t, rc2GuardRouter(g, user, true), "/res/"+id))
	// Unauthenticated (or malformed user ID) callers are rejected.
	assert.Equal(t, http.StatusUnauthorized, rc2GuardCode(t, rc2GuardRouter(g, "", false), "/wf/"+id))
	assert.Equal(t, http.StatusUnauthorized, rc2GuardCode(t, rc2GuardRouter(g, uuid.Nil.String(), false), "/wf/"+id))
}

func TestRC2_OwnershipGuard_LookupOutcomes(t *testing.T) {
	t.Parallel()
	owner := uuid.New()
	other := uuid.New()
	ownerStr := owner.String()
	boom := errors.New("db exploded")
	notFound := errors.New("record not found")

	wfOwned := &storagemodels.WorkflowModel{ID: uuid.New(), CreatedBy: &owner}
	wfID := wfOwned.ID

	type tc struct {
		name  string
		guard *OwnershipGuard
		path  string
		user  uuid.UUID
		admin bool
		want  int
	}
	log := rc2Logger()
	wfRepo := func(wf *storagemodels.WorkflowModel, err error) repository.WorkflowRepository {
		return &rc2WorkflowLookup{wf: wf, err: err}
	}
	guard := func(wf repository.WorkflowRepository, ex repository.ExecutionRepository, tr repository.TriggerRepository, res repository.ResourceRepository) *OwnershipGuard {
		return NewOwnershipGuard(wf, ex, tr, nil, res, log)
	}
	id := "/" + uuid.NewString()

	cases := []tc{
		// Workflows.
		{"wf owner", guard(wfRepo(wfOwned, nil), nil, nil, nil), "/wf" + id, owner, false, 204},
		{"wf other", guard(wfRepo(wfOwned, nil), nil, nil, nil), "/wf" + id, other, false, 404},
		{"wf nil record", guard(wfRepo(nil, nil), nil, nil, nil), "/wf" + id, owner, false, 404},
		{"wf not found", guard(wfRepo(nil, notFound), nil, nil, nil), "/wf" + id, owner, false, 404},
		{"wf error", guard(wfRepo(nil, boom), nil, nil, nil), "/wf" + id, owner, false, 500},
		{"wf unowned admin", guard(wfRepo(&storagemodels.WorkflowModel{}, nil), nil, nil, nil), "/wf" + id, other, true, 204},
		{"wf unowned user", guard(wfRepo(&storagemodels.WorkflowModel{}, nil), nil, nil, nil), "/wf" + id, owner, false, 404},

		// Executions.
		{"exec error", guard(nil, &rc2ExecutionLookup{err: boom}, nil, nil), "/ex" + id, owner, false, 500},
		{"exec not found", guard(nil, &rc2ExecutionLookup{err: sql.ErrNoRows}, nil, nil), "/ex" + id, owner, false, 404},
		{"exec nil", guard(nil, &rc2ExecutionLookup{}, nil, nil), "/ex" + id, owner, false, 404},
		{"exec admin", guard(nil, &rc2ExecutionLookup{exec: &storagemodels.ExecutionModel{}}, nil, nil), "/ex" + id, other, true, 204},
		{"exec creator", guard(nil, &rc2ExecutionLookup{exec: &storagemodels.ExecutionModel{CreatedBy: &owner}}, nil, nil), "/ex" + id, owner, false, 204},
		{"exec via workflow", guard(wfRepo(wfOwned, nil), &rc2ExecutionLookup{exec: &storagemodels.ExecutionModel{WorkflowID: &wfID}}, nil, nil), "/ex" + id, owner, false, 204},
		{"exec via other workflow", guard(wfRepo(wfOwned, nil), &rc2ExecutionLookup{exec: &storagemodels.ExecutionModel{WorkflowID: &wfID}}, nil, nil), "/ex" + id, other, false, 404},
		{"exec inline other", guard(nil, &rc2ExecutionLookup{exec: &storagemodels.ExecutionModel{CreatedBy: &owner}}, nil, nil), "/ex" + id, other, false, 404},
		{"exec orphan", guard(nil, &rc2ExecutionLookup{exec: &storagemodels.ExecutionModel{}}, nil, nil), "/ex" + id, owner, false, 404},

		// Triggers.
		{"trigger error", guard(nil, nil, &rc2TriggerLookup{err: boom}, nil), "/tr" + id, owner, false, 500},
		{"trigger not found", guard(nil, nil, &rc2TriggerLookup{err: pkgmodels.ErrTriggerNotFound}, nil), "/tr" + id, owner, false, 404},
		{"trigger nil", guard(nil, nil, &rc2TriggerLookup{}, nil), "/tr" + id, owner, false, 404},
		{"trigger admin", guard(nil, nil, &rc2TriggerLookup{tr: &storagemodels.TriggerModel{}}, nil), "/tr" + id, other, true, 204},
		{"trigger owner", guard(wfRepo(wfOwned, nil), nil, &rc2TriggerLookup{tr: &storagemodels.TriggerModel{WorkflowID: wfID}}, nil), "/tr" + id, owner, false, 204},
		{"trigger other", guard(wfRepo(wfOwned, nil), nil, &rc2TriggerLookup{tr: &storagemodels.TriggerModel{WorkflowID: wfID}}, nil), "/tr" + id, other, false, 404},

		// Resources.
		{"resource owner", guard(nil, nil, nil, &rc2ResourceLookup{res: pkgmodels.NewFileStorageResource(ownerStr, "r")}), "/res" + id, owner, false, 204},
		{"resource other", guard(nil, nil, nil, &rc2ResourceLookup{res: pkgmodels.NewFileStorageResource(ownerStr, "r")}), "/res" + id, other, false, 404},
		{"resource not found", guard(nil, nil, nil, &rc2ResourceLookup{err: pkgmodels.ErrResourceNotFound}), "/res" + id, owner, false, 404},
		{"resource error", guard(nil, nil, nil, &rc2ResourceLookup{err: boom}), "/res" + id, owner, false, 500},
		{"resource nil", guard(nil, nil, nil, &rc2ResourceLookup{}), "/res" + id, owner, false, 404},
	}
	for _, c := range cases {
		r := rc2GuardRouter(c.guard, c.user.String(), c.admin)
		assert.Equal(t, c.want, rc2GuardCode(t, r, c.path), c.name)
	}

	// Malformed IDs are passed through to the handler.
	r := rc2GuardRouter(guard(wfRepo(nil, boom), nil, nil, nil), owner.String(), false)
	assert.Equal(t, http.StatusNoContent, rc2GuardCode(t, r, "/wf/not-a-uuid"))
}

func TestRC2_OwnershipGuard_FilesAgainstDatabase(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	aliceID := uuid.MustParse(alice)
	ctx := context.Background()

	wfID := env.createWorkflow(t, env.router(alice, false, true), map[string]any{"name": "wf"})["id"].(string)
	wfUUID := uuid.MustParse(wfID)
	own := &storagemodels.FileModel{ID: uuid.New(), StorageID: "default", Name: "a", Path: "a", MimeType: "image/png", AccessScope: "workflow", CreatedBy: &aliceID}
	viaWorkflow := &storagemodels.FileModel{ID: uuid.New(), StorageID: "default", Name: "b", Path: "b", MimeType: "image/png", AccessScope: "workflow", WorkflowID: &wfUUID}
	orphan := &storagemodels.FileModel{ID: uuid.New(), StorageID: "default", Name: "c", Path: "c", MimeType: "image/png", AccessScope: "workflow"}
	for _, f := range []*storagemodels.FileModel{own, viaWorkflow, orphan} {
		require.NoError(t, env.fileRepo.Create(ctx, f))
	}

	ra := rc2GuardRouter(env.guard, alice, false)
	rb := rc2GuardRouter(env.guard, bob, false)
	assert.Equal(t, http.StatusNoContent, rc2GuardCode(t, ra, "/file/"+own.ID.String()))
	assert.Equal(t, http.StatusNoContent, rc2GuardCode(t, ra, "/file/"+viaWorkflow.ID.String()), "files of an owned workflow")
	assert.Equal(t, http.StatusNotFound, rc2GuardCode(t, rb, "/file/"+own.ID.String()))
	assert.Equal(t, http.StatusNotFound, rc2GuardCode(t, rb, "/file/"+viaWorkflow.ID.String()))
	assert.Equal(t, http.StatusNotFound, rc2GuardCode(t, ra, "/file/"+orphan.ID.String()), "unowned files are admin-only")
	assert.Equal(t, http.StatusNoContent, rc2GuardCode(t, rc2GuardRouter(env.guard, bob, true), "/file/"+orphan.ID.String()))
	assert.Equal(t, http.StatusNotFound, rc2GuardCode(t, ra, "/file/"+uuid.NewString()))

	require.NoError(t, env.db.Close())
	assert.Equal(t, http.StatusInternalServerError, rc2GuardCode(t, ra, "/file/"+own.ID.String()))
}
