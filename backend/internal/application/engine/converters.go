package engine

import (
	"github.com/google/uuid"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// WorkflowModelToDomain converts storage WorkflowModel to domain Workflow
func WorkflowModelToDomain(wm *storagemodels.WorkflowModel) *models.Workflow {
	if wm == nil {
		return nil
	}

	workflow := &models.Workflow{
		ID:          wm.ID.String(),
		Name:        wm.Name,
		Description: wm.Description,
		Status:      models.WorkflowStatus(wm.Status),
		Variables:   make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   wm.CreatedAt,
		UpdatedAt:   wm.UpdatedAt,
	}

	// Convert Variables
	if wm.Variables != nil {
		workflow.Variables = map[string]interface{}(wm.Variables)
	}

	// Convert Metadata
	if wm.Metadata != nil {
		workflow.Metadata = map[string]interface{}(wm.Metadata)
	}

	// Convert Nodes
	workflow.Nodes = make([]*models.Node, 0, len(wm.Nodes))
	for _, nm := range wm.Nodes {
		workflow.Nodes = append(workflow.Nodes, NodeModelToDomain(nm))
	}

	// Convert Edges
	workflow.Edges = make([]*models.Edge, 0, len(wm.Edges))
	for _, em := range wm.Edges {
		workflow.Edges = append(workflow.Edges, EdgeModelToDomain(em))
	}

	return workflow
}

// NodeModelToDomain converts storage NodeModel to domain Node
func NodeModelToDomain(nm *storagemodels.NodeModel) *models.Node {
	if nm == nil {
		return nil
	}

	node := &models.Node{
		ID:     nm.NodeID, // Use logical ID
		Name:   nm.Name,
		Type:   nm.Type,
		Config: make(map[string]interface{}),
	}

	// Convert Config
	if nm.Config != nil {
		node.Config = map[string]interface{}(nm.Config)
	}

	// Convert Position if present
	if nm.Position != nil {
		posMap := map[string]interface{}(nm.Position)
		if x, ok := posMap["x"].(float64); ok {
			if y, ok := posMap["y"].(float64); ok {
				node.Position = &models.Position{X: x, Y: y}
			}
		}
	}

	return node
}

// EdgeModelToDomain converts storage EdgeModel to domain Edge
func EdgeModelToDomain(em *storagemodels.EdgeModel) *models.Edge {
	if em == nil {
		return nil
	}

	edge := &models.Edge{
		ID:   em.EdgeID,     // Use logical ID
		From: em.FromNodeID, // Use logical ID
		To:   em.ToNodeID,   // Use logical ID
	}

	// Convert condition if present
	if em.Condition != nil {
		if expr, ok := em.Condition["expression"].(string); ok {
			edge.Condition = expr
		}
	}

	return edge
}

// ExecutionModelToDomain converts storage ExecutionModel to domain Execution
func ExecutionModelToDomain(exm *storagemodels.ExecutionModel) *models.Execution {
	if exm == nil {
		return nil
	}

	exec := &models.Execution{
		ID:         exm.ID.String(),
		WorkflowID: exm.WorkflowID.String(),
		Status:     models.ExecutionStatus(exm.Status),
		Input:      make(map[string]interface{}),
		Output:     make(map[string]interface{}),
		Variables:  make(map[string]interface{}),
	}

	if exm.StartedAt != nil {
		exec.StartedAt = *exm.StartedAt
	}

	if exm.InputData != nil {
		exec.Input = exm.InputData
	}

	if exm.OutputData != nil {
		exec.Output = exm.OutputData
	}

	if exm.Variables != nil {
		exec.Variables = exm.Variables
	}

	if exm.CompletedAt != nil {
		exec.CompletedAt = exm.CompletedAt
	}

	if exm.Error != "" {
		exec.Error = exm.Error
	}

	// Convert NodeExecutions
	if len(exm.NodeExecutions) > 0 {
		exec.NodeExecutions = make([]*models.NodeExecution, len(exm.NodeExecutions))
		for i, ne := range exm.NodeExecutions {
			exec.NodeExecutions[i] = NodeExecutionModelToDomain(ne)
		}
	}

	return exec
}

// ExecutionDomainToModel converts domain Execution to storage ExecutionModel
func ExecutionDomainToModel(exec *models.Execution) *storagemodels.ExecutionModel {
	if exec == nil {
		return nil
	}

	exm := &storagemodels.ExecutionModel{
		Status:     string(exec.Status),
		InputData:  storagemodels.JSONBMap(exec.Input),
		OutputData: storagemodels.JSONBMap(exec.Output),
		Variables:  storagemodels.JSONBMap(exec.Variables),
		StartedAt:  &exec.StartedAt,
		Error:      exec.Error,
	}

	// Parse UUIDs
	if exec.ID != "" {
		if id, err := uuid.Parse(exec.ID); err == nil {
			exm.ID = id
		}
	}

	if exec.WorkflowID != "" {
		if wfID, err := uuid.Parse(exec.WorkflowID); err == nil {
			exm.WorkflowID = wfID
		}
	}

	if exec.CompletedAt != nil {
		exm.CompletedAt = exec.CompletedAt
	}

	// Convert node executions
	if len(exec.NodeExecutions) > 0 {
		exm.NodeExecutions = make([]*storagemodels.NodeExecutionModel, 0, len(exec.NodeExecutions))
		for _, ne := range exec.NodeExecutions {
			nem := NodeExecutionDomainToModel(ne)
			if nem != nil {
				exm.NodeExecutions = append(exm.NodeExecutions, nem)
			}
		}
	}

	return exm
}

// NodeExecutionModelToDomain converts storage NodeExecutionModel to domain NodeExecution
func NodeExecutionModelToDomain(nem *storagemodels.NodeExecutionModel) *models.NodeExecution {
	if nem == nil {
		return nil
	}

	ne := &models.NodeExecution{
		ID:             nem.ID.String(),
		ExecutionID:    nem.ExecutionID.String(),
		NodeID:         nem.NodeID.String(), // Will be replaced with logical ID by caller
		Status:         models.NodeExecutionStatus(nem.Status),
		Input:          make(map[string]interface{}),
		Output:         make(map[string]interface{}),
		Config:         make(map[string]interface{}),
		ResolvedConfig: make(map[string]interface{}),
		RetryCount:     nem.RetryCount,
	}

	// Copy input data
	if nem.InputData != nil {
		ne.Input = nem.InputData
	}

	// Copy output data
	if nem.OutputData != nil {
		ne.Output = nem.OutputData
	}

	// Copy config (original)
	if nem.Config != nil {
		ne.Config = nem.Config
	}

	// Copy resolved config
	if nem.ResolvedConfig != nil {
		ne.ResolvedConfig = nem.ResolvedConfig
	}

	// Copy started time
	if nem.StartedAt != nil {
		ne.StartedAt = *nem.StartedAt
	}

	// Copy completed time
	if nem.CompletedAt != nil {
		ne.CompletedAt = nem.CompletedAt
	}

	// Copy error
	if nem.Error != "" {
		ne.Error = nem.Error
	}

	return ne
}

// NodeExecutionDomainToModel converts domain NodeExecution to storage NodeExecutionModel
func NodeExecutionDomainToModel(ne *models.NodeExecution) *storagemodels.NodeExecutionModel {
	if ne == nil {
		return nil
	}

	nem := &storagemodels.NodeExecutionModel{
		Status:         string(ne.Status),
		InputData:      storagemodels.JSONBMap(ne.Input),
		OutputData:     storagemodels.JSONBMap(ne.Output),
		Config:         storagemodels.JSONBMap(ne.Config),
		ResolvedConfig: storagemodels.JSONBMap(ne.ResolvedConfig),
		RetryCount:     ne.RetryCount,
		Error:          ne.Error,
	}

	// Parse UUIDs
	if ne.ID != "" {
		if id, err := uuid.Parse(ne.ID); err == nil {
			nem.ID = id
		} else {
			nem.ID = uuid.New() // Generate new ID if parsing fails
		}
	} else {
		nem.ID = uuid.New() // Generate new ID if empty
	}

	if ne.ExecutionID != "" {
		if execID, err := uuid.Parse(ne.ExecutionID); err == nil {
			nem.ExecutionID = execID
		}
	}

	if ne.NodeID != "" {
		// NodeID in domain is logical ID (string), but we need the UUID from the workflow
		// This is a bit tricky - we'll need to convert it properly
		// For now, try to parse it as UUID, if it fails, we'll need workflow context
		if nodeID, err := uuid.Parse(ne.NodeID); err == nil {
			nem.NodeID = nodeID
		}
	}

	// Copy timestamps
	if !ne.StartedAt.IsZero() {
		nem.StartedAt = &ne.StartedAt
	}
	if ne.CompletedAt != nil && !ne.CompletedAt.IsZero() {
		nem.CompletedAt = ne.CompletedAt
	}

	return nem
}
