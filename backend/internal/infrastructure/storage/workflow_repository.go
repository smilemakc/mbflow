package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/uptrace/bun"
)

// Ensure WorkflowRepository implements the interface
var _ repository.WorkflowRepository = (*WorkflowRepository)(nil)

// WorkflowRepository implements repository.WorkflowRepository using Bun ORM
type WorkflowRepository struct {
	db *bun.DB
}

// NewWorkflowRepository creates a new WorkflowRepository
func NewWorkflowRepository(db *bun.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// Create creates a new workflow with its nodes and edges
func (r *WorkflowRepository) Create(ctx context.Context, workflow *models.WorkflowModel) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// 1. Create workflow
		if _, err := tx.NewInsert().Model(workflow).Exec(ctx); err != nil {
			return fmt.Errorf("failed to create workflow: %w", err)
		}

		// 2. Create nodes
		if len(workflow.Nodes) > 0 {
			for _, node := range workflow.Nodes {
				node.WorkflowID = workflow.ID
				if node.ID == uuid.Nil {
					node.ID = uuid.New()
				}
			}
			if _, err := tx.NewInsert().Model(&workflow.Nodes).Exec(ctx); err != nil {
				return fmt.Errorf("failed to create nodes: %w", err)
			}
		}

		// 3. Create edges
		if len(workflow.Edges) > 0 {
			for _, edge := range workflow.Edges {
				edge.WorkflowID = workflow.ID
				if edge.ID == uuid.Nil {
					edge.ID = uuid.New()
				}
			}
			if _, err := tx.NewInsert().Model(&workflow.Edges).Exec(ctx); err != nil {
				return fmt.Errorf("failed to create edges: %w", err)
			}
		}

		return nil
	})
}

// Update updates an existing workflow using smart merge strategy
// - Existing nodes (by node_id): preserve UUID, update fields
// - New nodes: create with new UUID
// - Missing nodes: delete from DB
func (r *WorkflowRepository) Update(ctx context.Context, workflow *models.WorkflowModel) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// 1. Update workflow metadata
		workflow.UpdatedAt = time.Now()
		_, err := tx.NewUpdate().
			Model(workflow).
			Column("name", "description", "version", "status", "variables", "metadata", "updated_at").
			Where("id = ?", workflow.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update workflow: %w", err)
		}

		// 2. Sync nodes (smart merge)
		if err := r.syncNodes(ctx, tx, workflow.ID, workflow.Nodes); err != nil {
			return fmt.Errorf("failed to sync nodes: %w", err)
		}

		// 3. Sync edges (smart merge)
		if err := r.syncEdges(ctx, tx, workflow.ID, workflow.Edges); err != nil {
			return fmt.Errorf("failed to sync edges: %w", err)
		}

		return nil
	})
}

// syncNodes performs a smart merge of nodes
func (r *WorkflowRepository) syncNodes(
	ctx context.Context,
	tx bun.Tx,
	workflowID uuid.UUID,
	nodes []*models.NodeModel,
) error {
	// 1. Get existing nodes from DB
	var existingNodes []*models.NodeModel
	err := tx.NewSelect().
		Model(&existingNodes).
		Where("workflow_id = ?", workflowID).
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// 2. Build lookup map: node_id -> existing NodeModel
	existingMap := make(map[string]*models.NodeModel)
	for _, node := range existingNodes {
		existingMap[node.NodeID] = node
	}

	// 3. Build incoming map
	incomingMap := make(map[string]*models.NodeModel)
	for _, node := range nodes {
		incomingMap[node.NodeID] = node
	}

	// 4. Update or Create nodes
	for _, incomingNode := range nodes {
		if existing, exists := existingMap[incomingNode.NodeID]; exists {
			// Node exists - UPDATE with preserved UUID
			incomingNode.ID = existing.ID // ⚡ Preserve UUID!
			incomingNode.CreatedAt = existing.CreatedAt
			incomingNode.WorkflowID = workflowID

			_, err := tx.NewUpdate().
				Model(incomingNode).
				Column("name", "type", "config", "position", "updated_at").
				Where("id = ?", existing.ID).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update node %s: %w", incomingNode.NodeID, err)
			}
		} else {
			// New node - CREATE with new UUID
			incomingNode.ID = uuid.New()
			incomingNode.WorkflowID = workflowID

			_, err := tx.NewInsert().
				Model(incomingNode).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to create node %s: %w", incomingNode.NodeID, err)
			}
		}
	}

	// 5. Delete removed nodes
	for nodeID, existing := range existingMap {
		if _, stillExists := incomingMap[nodeID]; !stillExists {
			_, err := tx.NewDelete().
				Model((*models.NodeModel)(nil)).
				Where("id = ?", existing.ID).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to delete node %s: %w", nodeID, err)
			}
		}
	}

	return nil
}

// syncEdges performs a smart merge of edges
func (r *WorkflowRepository) syncEdges(
	ctx context.Context,
	tx bun.Tx,
	workflowID uuid.UUID,
	edges []*models.EdgeModel,
) error {
	// 1. Get existing edges from DB
	var existingEdges []*models.EdgeModel
	err := tx.NewSelect().
		Model(&existingEdges).
		Where("workflow_id = ?", workflowID).
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// 2. Build lookup map: edge_id -> existing EdgeModel
	existingMap := make(map[string]*models.EdgeModel)
	for _, edge := range existingEdges {
		existingMap[edge.EdgeID] = edge
	}

	// 3. Build incoming map
	incomingMap := make(map[string]*models.EdgeModel)
	for _, edge := range edges {
		incomingMap[edge.EdgeID] = edge
	}

	// 4. Update or Create edges
	for _, incomingEdge := range edges {
		if existing, exists := existingMap[incomingEdge.EdgeID]; exists {
			// Edge exists - UPDATE with preserved UUID
			incomingEdge.ID = existing.ID // ⚡ Preserve UUID!
			incomingEdge.CreatedAt = existing.CreatedAt
			incomingEdge.WorkflowID = workflowID

			_, err := tx.NewUpdate().
				Model(incomingEdge).
				Column("from_node_id", "to_node_id", "condition", "updated_at").
				Where("id = ?", existing.ID).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update edge %s: %w", incomingEdge.EdgeID, err)
			}
		} else {
			// New edge - CREATE with new UUID
			incomingEdge.ID = uuid.New()
			incomingEdge.WorkflowID = workflowID

			_, err := tx.NewInsert().
				Model(incomingEdge).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to create edge %s: %w", incomingEdge.EdgeID, err)
			}
		}
	}

	// 5. Delete removed edges
	for edgeID, existing := range existingMap {
		if _, stillExists := incomingMap[edgeID]; !stillExists {
			_, err := tx.NewDelete().
				Model((*models.EdgeModel)(nil)).
				Where("id = ?", existing.ID).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to delete edge %s: %w", edgeID, err)
			}
		}
	}

	return nil
}

// Delete soft-deletes a workflow
func (r *WorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Assuming WorkflowModel has a DeletedAt field for soft deletes
	// If not, this will be a hard delete
	_, err := r.db.NewDelete().
		Model((*models.WorkflowModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// HardDelete permanently deletes a workflow
func (r *WorkflowRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Cascade delete handled by foreign keys, but we can be explicit
		_, err := tx.NewDelete().
			Model((*models.WorkflowModel)(nil)).
			Where("id = ?", id).
			Exec(ctx)
		return err
	})
}

// FindByID retrieves a workflow by ID
func (r *WorkflowRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.WorkflowModel, error) {
	workflow := &models.WorkflowModel{}
	err := r.db.NewSelect().
		Model(workflow).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

// FindByIDWithRelations retrieves a workflow with all its relations (nodes, edges, triggers)
func (r *WorkflowRepository) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*models.WorkflowModel, error) {
	workflow := &models.WorkflowModel{}
	err := r.db.NewSelect().
		Model(workflow).
		Relation("Nodes").
		Relation("Edges").
		Where("w.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

// FindByName retrieves a workflow by name and version
func (r *WorkflowRepository) FindByName(ctx context.Context, name string, version int) (*models.WorkflowModel, error) {
	workflow := &models.WorkflowModel{}
	err := r.db.NewSelect().
		Model(workflow).
		Where("name = ? AND version = ?", name, version).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

// FindAll retrieves all workflows with pagination
func (r *WorkflowRepository) FindAll(ctx context.Context, limit, offset int) ([]*models.WorkflowModel, error) {
	var workflows []*models.WorkflowModel
	err := r.db.NewSelect().
		Model(&workflows).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return workflows, nil
}

// FindByStatus retrieves workflows by status with pagination
func (r *WorkflowRepository) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*models.WorkflowModel, error) {
	var workflows []*models.WorkflowModel
	err := r.db.NewSelect().
		Model(&workflows).
		Where("status = ?", status).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return workflows, nil
}

// Count returns the total count of workflows
func (r *WorkflowRepository) Count(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.WorkflowModel)(nil)).
		Count(ctx)
	return count, err
}

// CountByStatus returns the count of workflows by status
func (r *WorkflowRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.WorkflowModel)(nil)).
		Where("status = ?", status).
		Count(ctx)
	return count, err
}

// CreateNode creates a new node for a workflow
func (r *WorkflowRepository) CreateNode(ctx context.Context, node *models.NodeModel) error {
	if node.ID == uuid.Nil {
		node.ID = uuid.New()
	}
	_, err := r.db.NewInsert().Model(node).Exec(ctx)
	return err
}

// UpdateNode updates an existing node by its logical ID
func (r *WorkflowRepository) UpdateNode(ctx context.Context, node *models.NodeModel) error {
	_, err := r.db.NewUpdate().
		Model(node).
		Column("name", "type", "config", "position", "updated_at").
		Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).
		Exec(ctx)
	return err
}

// DeleteNode deletes a node by its UUID
func (r *WorkflowRepository) DeleteNode(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.NodeModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// FindNodeByID retrieves a node by its UUID
func (r *WorkflowRepository) FindNodeByID(ctx context.Context, id uuid.UUID) (*models.NodeModel, error) {
	node := &models.NodeModel{}
	err := r.db.NewSelect().
		Model(node).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// FindNodesByWorkflowID retrieves all nodes for a workflow
func (r *WorkflowRepository) FindNodesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.NodeModel, error) {
	var nodes []*models.NodeModel
	err := r.db.NewSelect().
		Model(&nodes).
		Where("workflow_id = ?", workflowID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// CreateEdge creates a new edge for a workflow
func (r *WorkflowRepository) CreateEdge(ctx context.Context, edge *models.EdgeModel) error {
	if edge.ID == uuid.Nil {
		edge.ID = uuid.New()
	}
	_, err := r.db.NewInsert().Model(edge).Exec(ctx)
	return err
}

// UpdateEdge updates an existing edge by its logical ID
func (r *WorkflowRepository) UpdateEdge(ctx context.Context, edge *models.EdgeModel) error {
	_, err := r.db.NewUpdate().
		Model(edge).
		Column("from_node_id", "to_node_id", "condition", "updated_at").
		Where("workflow_id = ? AND edge_id = ?", edge.WorkflowID, edge.EdgeID).
		Exec(ctx)
	return err
}

// DeleteEdge deletes an edge by its UUID
func (r *WorkflowRepository) DeleteEdge(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.EdgeModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// FindEdgeByID retrieves an edge by its UUID
func (r *WorkflowRepository) FindEdgeByID(ctx context.Context, id uuid.UUID) (*models.EdgeModel, error) {
	edge := &models.EdgeModel{}
	err := r.db.NewSelect().
		Model(edge).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return edge, nil
}

// FindEdgesByWorkflowID retrieves all edges for a workflow
func (r *WorkflowRepository) FindEdgesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.EdgeModel, error) {
	var edges []*models.EdgeModel
	err := r.db.NewSelect().
		Model(&edges).
		Where("workflow_id = ?", workflowID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

// ValidateDAG validates that the workflow forms a valid DAG (no cycles)
func (r *WorkflowRepository) ValidateDAG(ctx context.Context, workflowID uuid.UUID) error {
	// Get all edges for the workflow
	edges, err := r.FindEdgesByWorkflowID(ctx, workflowID)
	if err != nil {
		return err
	}

	// Build adjacency list using logical node IDs
	graph := make(map[string][]string)
	for _, edge := range edges {
		graph[edge.FromNodeID] = append(graph[edge.FromNodeID], edge.ToNodeID)
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		for _, neighbor := range graph[nodeID] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range graph {
		if !visited[nodeID] {
			if hasCycle(nodeID) {
				return fmt.Errorf("cycle detected in workflow DAG")
			}
		}
	}

	return nil
}
