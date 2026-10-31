package storage

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

// rcCreateFileResource creates a file storage resource owned by ownerID and returns its UUID.
func rcCreateFileResource(t *testing.T, db bun.IDB, ownerID, name string) uuid.UUID {
	t.Helper()
	fs := rcNewFileStorage(ownerID, name)
	require.NoError(t, NewResourceRepository(db).Create(context.Background(), fs))
	return uuid.MustParse(fs.ID)
}

func rcWorkflow(name string, createdBy *uuid.UUID) *models.WorkflowModel {
	return &models.WorkflowModel{
		Name:      name,
		Status:    "draft",
		Version:   1,
		CreatedBy: createdBy,
		Nodes: []*models.NodeModel{
			{NodeID: "a", Name: "A", Type: "http", Config: models.JSONBMap{"url": "x"}},
			{NodeID: "b", Name: "B", Type: "transform"},
			{NodeID: "c", Name: "C", Type: "transform"},
		},
		Edges: []*models.EdgeModel{
			{EdgeID: "a-b", FromNodeID: "a", ToNodeID: "b"},
			{EdgeID: "b-c", FromNodeID: "b", ToNodeID: "c"},
		},
	}
}

func TestRC_WorkflowRepository_CreateWithResourcesAndRelations(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewWorkflowRepository(db)
	owner := createCredentialsTestUser(t, db)
	resID := rcCreateFileResource(t, db, owner, "files")

	wf := rcWorkflow("with-resources", nil)
	wf.Resources = []*models.WorkflowResourceModel{{ResourceID: resID, Alias: "files", AccessType: "write"}}
	require.NoError(t, repo.Create(ctx, wf))
	assert.False(t, wf.Resources[0].AssignedAt.IsZero())

	got, err := repo.FindByIDWithRelations(ctx, wf.ID)
	require.NoError(t, err)
	assert.Len(t, got.Nodes, 3)
	assert.Len(t, got.Edges, 2)
	require.Len(t, got.Resources, 1)
	assert.Equal(t, "files", got.Resources[0].Alias)
	require.NotNil(t, got.Resources[0].Resource)
	assert.Equal(t, "files", got.Resources[0].Resource.Name)

	domain := models.WorkflowModelToDomain(got)
	require.Len(t, domain.Resources, 1)
	assert.Equal(t, "file_storage", domain.Resources[0].ResourceType)

	// node insert failure (duplicate logical node id) aborts the whole workflow
	bad := rcWorkflow("bad-nodes", nil)
	bad.Nodes = append(bad.Nodes, &models.NodeModel{NodeID: "a", Name: "dup", Type: "http"})
	assert.Error(t, repo.Create(ctx, bad))
	_, err = repo.FindByID(ctx, bad.ID)
	assert.Error(t, err, "workflow insert must be rolled back")

	// edge referencing an unknown node violates the FK
	badEdge := rcWorkflow("bad-edges", nil)
	badEdge.Edges = append(badEdge.Edges, &models.EdgeModel{EdgeID: "x", FromNodeID: "a", ToNodeID: "missing"})
	assert.Error(t, repo.Create(ctx, badEdge))

	// resource referencing an unknown resource violates the FK
	badRes := rcWorkflow("bad-res", nil)
	badRes.Resources = []*models.WorkflowResourceModel{{ResourceID: uuid.New(), Alias: "x", AccessType: "read"}}
	assert.Error(t, repo.Create(ctx, badRes))

	// HardDelete removes the workflow and cascades
	require.NoError(t, repo.HardDelete(ctx, wf.ID))
	_, err = repo.FindByID(ctx, wf.ID)
	assert.Error(t, err)
	nodes, err := repo.FindNodesByWorkflowID(ctx, wf.ID)
	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestRC_WorkflowRepository_UpdateSyncsEdgesAndResources(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewWorkflowRepository(db)
	owner := createCredentialsTestUser(t, db)
	r1 := rcCreateFileResource(t, db, owner, "r1")
	r2 := rcCreateFileResource(t, db, owner, "r2")
	r3 := rcCreateFileResource(t, db, owner, "r3")

	wf := rcWorkflow("sync", nil)
	wf.Resources = []*models.WorkflowResourceModel{
		{ResourceID: r1, Alias: "one", AccessType: "read"},
		{ResourceID: r2, Alias: "two", AccessType: "read"},
	}
	require.NoError(t, repo.Create(ctx, wf))

	before, err := repo.FindEdgesByWorkflowID(ctx, wf.ID)
	require.NoError(t, err)
	ids := map[string]uuid.UUID{}
	for _, e := range before {
		ids[e.EdgeID] = e.ID
	}

	update := &models.WorkflowModel{
		ID:      wf.ID,
		Name:    "sync-v2",
		Status:  "active",
		Version: 2,
		Nodes: []*models.NodeModel{
			{NodeID: "a", Name: "A2", Type: "http", Config: models.JSONBMap{}},
			{NodeID: "b", Name: "B", Type: "transform", Config: models.JSONBMap{}},
			{NodeID: "c", Name: "C", Type: "transform", Config: models.JSONBMap{}},
		},
		Edges: []*models.EdgeModel{
			// existing edge rewired and given a condition, loop and handle
			{EdgeID: "a-b", FromNodeID: "a", ToNodeID: "c", Condition: models.JSONBMap{"expression": "x > 1"},
				SourceHandle: "true", Loop: models.JSONBMap{"max_iterations": 3}},
			// new edge
			{EdgeID: "a-b2", FromNodeID: "a", ToNodeID: "b"},
			// "b-c" removed
		},
		Resources: []*models.WorkflowResourceModel{
			{ResourceID: r1, Alias: "one-renamed", AccessType: "admin"},
			{ResourceID: r3, Alias: "three", AccessType: "write"},
			// r2 removed
		},
	}
	require.NoError(t, repo.Update(ctx, update))

	got, err := repo.FindByIDWithRelations(ctx, wf.ID)
	require.NoError(t, err)
	assert.Equal(t, "sync-v2", got.Name)
	assert.Equal(t, 2, got.Version)

	edges := map[string]*models.EdgeModel{}
	for _, e := range got.Edges {
		edges[e.EdgeID] = e
	}
	require.Len(t, edges, 2)
	require.Contains(t, edges, "a-b")
	require.Contains(t, edges, "a-b2")
	assert.Equal(t, ids["a-b"], edges["a-b"].ID, "existing edge keeps its UUID")
	assert.Equal(t, "c", edges["a-b"].ToNodeID)
	assert.Equal(t, "x > 1", edges["a-b"].Condition["expression"])
	assert.Equal(t, "true", edges["a-b"].SourceHandle, "source handle of an existing edge must be updated")
	require.NotNil(t, edges["a-b"].Loop, "loop config of an existing edge must be updated")
	assert.EqualValues(t, 3, edges["a-b"].Loop["max_iterations"])

	assert.Greater(t, edges["a-b"].UpdatedAt.Year(), 2000, "updated_at of a synced edge must be refreshed")
	for _, n := range got.Nodes {
		assert.Greater(t, n.UpdatedAt.Year(), 2000, "updated_at of synced node %s must be refreshed", n.NodeID)
	}

	res := map[uuid.UUID]*models.WorkflowResourceModel{}
	for _, r := range got.Resources {
		res[r.ResourceID] = r
	}
	require.Len(t, res, 2)
	assert.Equal(t, "one-renamed", res[r1].Alias)
	assert.Equal(t, "admin", res[r1].AccessType)
	assert.Equal(t, "three", res[r3].Alias)
	assert.NotContains(t, res, r2)

	// a failing edge sync (self reference) rolls back the whole update
	broken := &models.WorkflowModel{
		ID: wf.ID, Name: "broken", Status: "active", Version: 3,
		Nodes: update.Nodes,
		Edges: []*models.EdgeModel{{EdgeID: "self", FromNodeID: "a", ToNodeID: "a"}},
	}
	assert.Error(t, repo.Update(ctx, broken))
	got, err = repo.FindByID(ctx, wf.ID)
	require.NoError(t, err)
	assert.Equal(t, "sync-v2", got.Name)

	// a failing edge update (unknown target node) is reported
	brokenUpd := &models.WorkflowModel{
		ID: wf.ID, Name: "broken", Status: "active", Version: 3,
		Nodes: update.Nodes,
		Edges: []*models.EdgeModel{{EdgeID: "a-b", FromNodeID: "a", ToNodeID: "zzz"}},
	}
	assert.Error(t, repo.Update(ctx, brokenUpd))

	// failing resource sync (alias conflict / bad access type / unknown resource)
	brokenRes := &models.WorkflowModel{
		ID: wf.ID, Name: "broken", Status: "active", Version: 3,
		Nodes: update.Nodes, Edges: update.Edges,
		Resources: []*models.WorkflowResourceModel{{ResourceID: r1, Alias: "one", AccessType: "bogus"}},
	}
	assert.Error(t, repo.Update(ctx, brokenRes))
	brokenRes.Resources = []*models.WorkflowResourceModel{{ResourceID: uuid.New(), Alias: "nope", AccessType: "read"}}
	assert.Error(t, repo.Update(ctx, brokenRes))

	// invalid workflow status fails the first statement
	assert.Error(t, repo.Update(ctx, &models.WorkflowModel{ID: wf.ID, Name: "x", Status: "bogus", Version: 1}))
}

func TestRC_WorkflowRepository_FiltersAndCounts(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewWorkflowRepository(db)
	owner := uuid.MustParse(createCredentialsTestUser(t, db))
	other := uuid.MustParse(createCredentialsTestUser(t, db))

	mk := func(name, status string, by *uuid.UUID) {
		wf := rcWorkflow(name, by)
		wf.Status = status
		require.NoError(t, repo.Create(ctx, wf))
	}
	mk("o-draft", "draft", &owner)
	mk("o-active", "active", &owner)
	mk("x-active", "active", &other)
	mk("unowned", "active", nil)

	active := "active"
	empty := ""
	cases := []struct {
		name    string
		filters repository.WorkflowFilters
		want    int
	}{
		{"none", repository.WorkflowFilters{}, 4},
		{"empty status ignored", repository.WorkflowFilters{Status: &empty}, 4},
		{"status", repository.WorkflowFilters{Status: &active}, 3},
		{"owner", repository.WorkflowFilters{CreatedBy: &owner}, 2},
		{"owner+unowned", repository.WorkflowFilters{CreatedBy: &owner, IncludeUnowned: true}, 3},
		{"owner+status", repository.WorkflowFilters{CreatedBy: &owner, Status: &active}, 1},
	}
	for _, c := range cases {
		list, err := repo.FindAllWithFilters(ctx, c.filters, 10, 0)
		require.NoError(t, err, c.name)
		assert.Len(t, list, c.want, c.name)
		for _, w := range list {
			assert.Len(t, w.Nodes, 3, "nodes are loaded for listings")
			assert.Len(t, w.Edges, 2, "edges are loaded for listings")
		}
		n, err := repo.CountWithFilters(ctx, c.filters)
		require.NoError(t, err, c.name)
		assert.Equal(t, c.want, n, c.name)
	}

	page, err := repo.FindAllWithFilters(ctx, repository.WorkflowFilters{}, 3, 2)
	require.NoError(t, err)
	assert.Len(t, page, 2)
}

func TestRC_WorkflowRepository_NodeAndEdgeCRUD(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewWorkflowRepository(db)

	wf := rcWorkflow("graph", nil)
	require.NoError(t, repo.Create(ctx, wf))

	node := &models.NodeModel{WorkflowID: wf.ID, NodeID: "d", Name: "D", Type: "llm", Config: models.JSONBMap{"k": "v"}}
	require.NoError(t, repo.CreateNode(ctx, node))
	require.NotEqual(t, uuid.Nil, node.ID)
	assert.Error(t, repo.CreateNode(ctx, &models.NodeModel{WorkflowID: wf.ID, NodeID: "d", Name: "dup", Type: "llm"}))

	node.Name = "D2"
	node.Config = models.JSONBMap{"k": "v2"}
	require.NoError(t, repo.UpdateNode(ctx, node))
	got, err := repo.FindNodeByID(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, "D2", got.Name)
	assert.Greater(t, got.UpdatedAt.Year(), 2000)
	assert.Equal(t, "v2", got.Config["k"])

	edge := &models.EdgeModel{WorkflowID: wf.ID, EdgeID: "c-d", FromNodeID: "c", ToNodeID: "d"}
	require.NoError(t, repo.CreateEdge(ctx, edge))
	require.NotEqual(t, uuid.Nil, edge.ID)
	assert.Error(t, repo.CreateEdge(ctx, &models.EdgeModel{WorkflowID: wf.ID, EdgeID: "bad", FromNodeID: "c", ToNodeID: "nope"}))

	edge.FromNodeID = "a"
	edge.Condition = models.JSONBMap{"expression": "ok"}
	edge.SourceHandle = "out"
	edge.Loop = models.JSONBMap{"max_iterations": 2}
	require.NoError(t, repo.UpdateEdge(ctx, edge))
	gotEdge, err := repo.FindEdgeByID(ctx, edge.ID)
	require.NoError(t, err)
	assert.Equal(t, "a", gotEdge.FromNodeID)
	assert.Equal(t, "ok", gotEdge.Condition["expression"])
	assert.Equal(t, "out", gotEdge.SourceHandle)
	assert.EqualValues(t, 2, gotEdge.Loop["max_iterations"])
	assert.Greater(t, gotEdge.UpdatedAt.Year(), 2000)

	// a->d plus d->a forms a cycle
	require.NoError(t, repo.CreateEdge(ctx, &models.EdgeModel{WorkflowID: wf.ID, EdgeID: "d-a", FromNodeID: "d", ToNodeID: "a"}))
	assert.Error(t, repo.ValidateDAG(ctx, wf.ID))

	require.NoError(t, repo.DeleteEdge(ctx, edge.ID))
	_, err = repo.FindEdgeByID(ctx, edge.ID)
	assert.Error(t, err)

	require.NoError(t, repo.DeleteNode(ctx, node.ID))
	_, err = repo.FindNodeByID(ctx, node.ID)
	assert.Error(t, err)
	// deleting node d cascades to its remaining edges
	edges, err := repo.FindEdgesByWorkflowID(ctx, wf.ID)
	require.NoError(t, err)
	assert.Len(t, edges, 2)
	require.NoError(t, repo.ValidateDAG(ctx, wf.ID))
}

func TestRC_WorkflowRepository_ResourceAssignments(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewWorkflowRepository(db)
	owner := createCredentialsTestUser(t, db)
	ownerID := uuid.MustParse(owner)

	wf1 := rcWorkflow("wf1", nil)
	wf2 := rcWorkflow("wf2", nil)
	require.NoError(t, repo.Create(ctx, wf1))
	require.NoError(t, repo.Create(ctx, wf2))
	r1 := rcCreateFileResource(t, db, owner, "r1")
	r2 := rcCreateFileResource(t, db, owner, "r2")

	require.NoError(t, repo.AssignResource(ctx, wf1.ID, &models.WorkflowResourceModel{ResourceID: r1, Alias: "b-alias", AccessType: "read"}, &ownerID))
	require.NoError(t, repo.AssignResource(ctx, wf1.ID, &models.WorkflowResourceModel{ResourceID: r2, Alias: "a-alias", AccessType: "read"}, nil))
	require.NoError(t, repo.AssignResource(ctx, wf2.ID, &models.WorkflowResourceModel{ResourceID: r1, Alias: "shared", AccessType: "read"}, nil))

	// re-assigning upserts alias and access type
	require.NoError(t, repo.AssignResource(ctx, wf1.ID, &models.WorkflowResourceModel{ResourceID: r1, Alias: "c-alias", AccessType: "write"}, nil))
	// invalid access type / unknown resource
	assert.Error(t, repo.AssignResource(ctx, wf1.ID, &models.WorkflowResourceModel{ResourceID: r2, Alias: "x", AccessType: "bogus"}, nil))
	assert.Error(t, repo.AssignResource(ctx, wf1.ID, &models.WorkflowResourceModel{ResourceID: uuid.New(), Alias: "y", AccessType: "read"}, nil))

	list, err := repo.GetWorkflowResources(ctx, wf1.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "a-alias", list[0].Alias, "ordered by alias")
	assert.Equal(t, "c-alias", list[1].Alias)
	assert.Equal(t, "write", list[1].AccessType)
	require.NotNil(t, list[1].AssignedBy)
	assert.Equal(t, ownerID, *list[1].AssignedBy, "upsert keeps the original assigner")

	exists, err := repo.ResourceExists(ctx, wf1.ID, r1)
	require.NoError(t, err)
	assert.True(t, exists)
	exists, err = repo.ResourceExists(ctx, wf1.ID, uuid.New())
	require.NoError(t, err)
	assert.False(t, exists)

	byAlias, err := repo.GetResourceByAlias(ctx, wf1.ID, "c-alias")
	require.NoError(t, err)
	assert.Equal(t, r1, byAlias.ResourceID)
	_, err = repo.GetResourceByAlias(ctx, wf1.ID, "missing")
	assert.ErrorIs(t, err, pkgmodels.ErrResourceNotFound)

	require.NoError(t, repo.UpdateResourceAlias(ctx, wf1.ID, r1, "renamed"))
	byAlias, err = repo.GetResourceByAlias(ctx, wf1.ID, "renamed")
	require.NoError(t, err)
	assert.Equal(t, r1, byAlias.ResourceID)
	assert.Error(t, repo.UpdateResourceAlias(ctx, wf1.ID, uuid.New(), "x"))
	assert.Error(t, repo.UpdateResourceAlias(ctx, wf1.ID, r1, "a-alias"), "alias must be unique per workflow")

	require.NoError(t, repo.UnassignResource(ctx, wf1.ID, r2))
	assert.ErrorIs(t, repo.UnassignResource(ctx, wf1.ID, r2), pkgmodels.ErrResourceNotFound)

	n, err := repo.UnassignResourceFromAllWorkflows(ctx, r1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	list, err = repo.GetWorkflowResources(ctx, wf2.ID)
	require.NoError(t, err)
	assert.Empty(t, list)
}
