package executor

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVariableBinder_NoParents tests binding when node has no parents
func TestVariableBinder_NoParents(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	// Create execution with global variables
	execution := createTestExecution(map[string]any{
		"global_var": "global_value",
	})

	// Create node with no parents
	node := createTestNode("test", nil)
	graph := createTestGraph([]domain.Node{node}, nil)

	// Bind inputs
	inputs, err := binder.BindInputs(node, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	// Variables should be empty (no parents)
	assert.Empty(t, inputs.Variables.All())

	// Global context should contain global variables
	globalVal, ok := inputs.GlobalContext.Get("global_var")
	assert.True(t, ok)
	assert.Equal(t, "global_value", globalVal)

	// Global context should be read-only
	assert.True(t, inputs.GlobalContext.IsReadOnly())
}

// TestVariableBinder_SingleParent tests binding with one parent node
func TestVariableBinder_SingleParent(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	// Create execution
	execution := createTestExecution(map[string]any{
		"global_var": "global",
	})

	// Create parent and child nodes
	parent := createTestNode("parent", nil)
	child := createTestNode("child", nil)

	// Set parent output
	parentOutput := map[string]any{
		"result": 42,
		"status": "success",
	}
	execution.SetNodeOutput(parent.ID(), parentOutput)

	// Create graph with edge from parent to child
	graph := createTestGraph(
		[]domain.Node{parent, child},
		[]testEdge{{parent.ID(), child.ID()}},
	)

	// Bind inputs for child
	inputs, err := binder.BindInputs(child, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	// Variables should contain entire parent output under parent node name
	parentData, ok := inputs.Variables.Get("parent")
	assert.True(t, ok, "parent data should exist")

	parentMap, ok := parentData.(map[string]any)
	assert.True(t, ok, "parent data should be a map")
	assert.Equal(t, 42, parentMap["result"])
	assert.Equal(t, "success", parentMap["status"])

	// Direct keys should not exist
	assert.NotContains(t, inputs.Variables.All(), "result")
	assert.NotContains(t, inputs.Variables.All(), "status")

	// Global context still accessible
	globalVal, ok := inputs.GlobalContext.Get("global_var")
	assert.True(t, ok)
	assert.Equal(t, "global", globalVal)
}

// TestVariableBinder_MultipleParents_NamespaceStrategy tests namespace collision strategy
func TestVariableBinder_MultipleParents_NamespaceStrategy(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(nil)

	// Create three parent nodes
	parent1 := createTestNode("parent1", nil)
	parent2 := createTestNode("parent2", nil)
	parent3 := createTestNode("parent3", nil)
	child := createTestNode("child", &domain.InputBindingConfig{
		AutoBind:          true,
		CollisionStrategy: domain.CollisionStrategyNamespaceByParent,
	})

	// Set outputs for all parents (same key "result")
	execution.SetNodeOutput(parent1.ID(), map[string]any{"result": 10})
	execution.SetNodeOutput(parent2.ID(), map[string]any{"result": 20})
	execution.SetNodeOutput(parent3.ID(), map[string]any{"result": 30})

	// Create graph with all parents connecting to child
	graph := createTestGraph(
		[]domain.Node{parent1, parent2, parent3, child},
		[]testEdge{
			{parent1.ID(), child.ID()},
			{parent2.ID(), child.ID()},
			{parent3.ID(), child.ID()},
		},
	)

	// Bind inputs
	inputs, err := binder.BindInputs(child, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	// Variables should contain entire outputs under parent names
	vars := inputs.Variables.All()

	parent1Data, ok := vars["parent1"].(map[string]any)
	require.True(t, ok, "parent1 should be a map")
	assert.Equal(t, 10, parent1Data["result"])

	parent2Data, ok := vars["parent2"].(map[string]any)
	require.True(t, ok, "parent2 should be a map")
	assert.Equal(t, 20, parent2Data["result"])

	parent3Data, ok := vars["parent3"].(map[string]any)
	require.True(t, ok, "parent3 should be a map")
	assert.Equal(t, 30, parent3Data["result"])

	// Original "result" key should not exist at top level
	assert.NotContains(t, vars, "result")
}

// TestVariableBinder_MultipleParents_CollectStrategy tests collect collision strategy
func TestVariableBinder_MultipleParents_CollectStrategy(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(nil)

	parent1 := createTestNode("parent1", nil)
	parent2 := createTestNode("parent2", nil)
	child := createTestNode("child", &domain.InputBindingConfig{
		AutoBind:          true,
		CollisionStrategy: domain.CollisionStrategyCollect,
	})

	// Both parents produce "result" key
	execution.SetNodeOutput(parent1.ID(), map[string]any{
		"result":  100,
		"unique1": "value1",
	})
	execution.SetNodeOutput(parent2.ID(), map[string]any{
		"result":  200,
		"unique2": "value2",
	})

	graph := createTestGraph(
		[]domain.Node{parent1, parent2, child},
		[]testEdge{
			{parent1.ID(), child.ID()},
			{parent2.ID(), child.ID()},
		},
	)

	inputs, err := binder.BindInputs(child, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	vars := inputs.Variables.All()

	// "result" should be collected into array
	resultValues, ok := vars["result"].([]any)
	require.True(t, ok, "result should be an array")
	assert.Len(t, resultValues, 2)
	assert.Contains(t, resultValues, 100)
	assert.Contains(t, resultValues, 200)

	// Unique keys should be preserved as single values
	assert.Equal(t, "value1", vars["unique1"])
	assert.Equal(t, "value2", vars["unique2"])
}

// TestVariableBinder_MultipleParents_ErrorStrategy tests error collision strategy
func TestVariableBinder_MultipleParents_ErrorStrategy(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(nil)

	parent1 := createTestNode("parent1", nil)
	parent2 := createTestNode("parent2", nil)
	child := createTestNode("child", &domain.InputBindingConfig{
		AutoBind:          true,
		CollisionStrategy: domain.CollisionStrategyError,
	})

	// Both parents produce "result" key - this should cause error
	execution.SetNodeOutput(parent1.ID(), map[string]any{"result": 100})
	execution.SetNodeOutput(parent2.ID(), map[string]any{"result": 200})

	graph := createTestGraph(
		[]domain.Node{parent1, parent2, child},
		[]testEdge{
			{parent1.ID(), child.ID()},
			{parent2.ID(), child.ID()},
		},
	)

	// Should return error due to collision
	_, err := binder.BindInputs(child, graph, execution)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "collision detected")
}

// TestVariableBinder_MultipleParents_ErrorStrategy_NoCollision tests error strategy with no collision
func TestVariableBinder_MultipleParents_ErrorStrategy_NoCollision(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(nil)

	parent1 := createTestNode("parent1", nil)
	parent2 := createTestNode("parent2", nil)
	child := createTestNode("child", &domain.InputBindingConfig{
		AutoBind:          true,
		CollisionStrategy: domain.CollisionStrategyError,
	})

	// Different keys - no collision
	execution.SetNodeOutput(parent1.ID(), map[string]any{"key1": 100})
	execution.SetNodeOutput(parent2.ID(), map[string]any{"key2": 200})

	graph := createTestGraph(
		[]domain.Node{parent1, parent2, child},
		[]testEdge{
			{parent1.ID(), child.ID()},
			{parent2.ID(), child.ID()},
		},
	)

	inputs, err := binder.BindInputs(child, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	// Both keys should be present
	vars := inputs.Variables.All()
	assert.Equal(t, 100, vars["key1"])
	assert.Equal(t, 200, vars["key2"])
}

// TestVariableBinder_ExplicitMappings tests custom variable mappings
func TestVariableBinder_ExplicitMappings(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(nil)

	parent := createTestNode("parent", nil)
	child := createTestNode("child", &domain.InputBindingConfig{
		AutoBind: false, // Disable auto-bind
		Mappings: map[string]string{
			"input_value":  "output_value",
			"input_status": "status",
		},
	})

	execution.SetNodeOutput(parent.ID(), map[string]any{
		"output_value": 42,
		"status":       "ok",
		"ignored":      "this should not appear",
	})

	graph := createTestGraph(
		[]domain.Node{parent, child},
		[]testEdge{{parent.ID(), child.ID()}},
	)

	inputs, err := binder.BindInputs(child, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	vars := inputs.Variables.All()

	// Mapped values should be present
	assert.Equal(t, 42, vars["input_value"])
	assert.Equal(t, "ok", vars["input_status"])

	// Original keys and ignored values should not be present
	assert.NotContains(t, vars, "output_value")
	assert.NotContains(t, vars, "status")
	assert.NotContains(t, vars, "ignored")
}

// TestVariableBinder_ExplicitMappings_WithNodeName tests mappings with node name prefix
func TestVariableBinder_ExplicitMappings_WithNodeName(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(nil)

	parent1 := createTestNode("parent1", nil)
	parent2 := createTestNode("parent2", nil)
	child := createTestNode("child", &domain.InputBindingConfig{
		AutoBind: false,
		Mappings: map[string]string{
			"value1": "parent1.result",
			"value2": "parent2.result",
		},
	})

	execution.SetNodeOutput(parent1.ID(), map[string]any{"result": 100})
	execution.SetNodeOutput(parent2.ID(), map[string]any{"result": 200})

	graph := createTestGraph(
		[]domain.Node{parent1, parent2, child},
		[]testEdge{
			{parent1.ID(), child.ID()},
			{parent2.ID(), child.ID()},
		},
	)

	inputs, err := binder.BindInputs(child, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	vars := inputs.Variables.All()
	assert.Equal(t, 100, vars["value1"])
	assert.Equal(t, 200, vars["value2"])
}

// TestVariableBinder_AutoBindWithMappings tests combining auto-bind and explicit mappings
func TestVariableBinder_AutoBindWithMappings(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(nil)

	parent := createTestNode("parent", nil)
	child := createTestNode("child", &domain.InputBindingConfig{
		AutoBind: true,
		Mappings: map[string]string{
			"renamed": "original",
		},
		CollisionStrategy: domain.CollisionStrategyNamespaceByParent,
	})

	execution.SetNodeOutput(parent.ID(), map[string]any{
		"original": "value1",
		"other":    "value2",
	})

	graph := createTestGraph(
		[]domain.Node{parent, child},
		[]testEdge{{parent.ID(), child.ID()}},
	)

	inputs, err := binder.BindInputs(child, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	vars := inputs.Variables.All()

	// Auto-bound values (entire output under parent name)
	parentData, ok := vars["parent"].(map[string]any)
	require.True(t, ok, "parent should be a map")
	assert.Equal(t, "value1", parentData["original"])
	assert.Equal(t, "value2", parentData["other"])

	// Explicitly mapped value (overrides auto-bind)
	assert.Equal(t, "value1", vars["renamed"])
}

// TestVariableBinder_InputSchemaValidation tests input schema validation
func TestVariableBinder_InputSchemaValidation(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(nil)

	parent := createTestNode("parent", nil)

	// Create child with input schema requiring specific fields
	childNode := createTestNode("child", &domain.InputBindingConfig{
		AutoBind:          true,
		CollisionStrategy: domain.CollisionStrategyNamespaceByParent,
	})

	// Set schema with required fields
	inputSchema := domain.NewVariableSchema()
	inputSchema.AddDefinition(&domain.VariableDefinition{
		Name:     "required_field",
		Required: true,
	})

	schema := &domain.NodeIOSchema{
		Inputs: inputSchema,
	}
	childNode.(*testNode).schema = schema

	// Parent output missing required field
	execution.SetNodeOutput(parent.ID(), map[string]any{
		"optional_field": "value",
	})

	graph := createTestGraph(
		[]domain.Node{parent, childNode},
		[]testEdge{{parent.ID(), childNode.ID()}},
	)

	// Should fail validation
	_, err := binder.BindInputs(childNode, graph, execution)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "input validation failed")
}

// TestVariableBinder_ComplexScenario tests a complex multi-parent scenario
func TestVariableBinder_ComplexScenario(t *testing.T) {
	binder := NewVariableBinder(NewConditionEvaluator(false))

	execution := createTestExecution(map[string]any{
		"config_value": "global_config",
	})

	// Three parents with overlapping keys
	parent1 := createTestNode("fetch_user", nil)
	parent2 := createTestNode("fetch_orders", nil)
	parent3 := createTestNode("fetch_settings", nil)

	child := createTestNode("aggregate", &domain.InputBindingConfig{
		AutoBind:          true,
		CollisionStrategy: domain.CollisionStrategyNamespaceByParent,
	})

	execution.SetNodeOutput(parent1.ID(), map[string]any{
		"id":   123,
		"name": "John Doe",
	})
	execution.SetNodeOutput(parent2.ID(), map[string]any{
		"id":    456,
		"total": 99.99,
	})
	execution.SetNodeOutput(parent3.ID(), map[string]any{
		"theme": "dark",
	})

	graph := createTestGraph(
		[]domain.Node{parent1, parent2, parent3, child},
		[]testEdge{
			{parent1.ID(), child.ID()},
			{parent2.ID(), child.ID()},
			{parent3.ID(), child.ID()},
		},
	)

	inputs, err := binder.BindInputs(child, graph, execution)

	require.NoError(t, err)
	require.NotNil(t, inputs)

	vars := inputs.Variables.All()

	// All variables should be stored under parent node names
	fetchUserData, ok := vars["fetch_user"].(map[string]any)
	require.True(t, ok, "fetch_user should be a map")
	assert.Equal(t, 123, fetchUserData["id"])
	assert.Equal(t, "John Doe", fetchUserData["name"])

	fetchOrdersData, ok := vars["fetch_orders"].(map[string]any)
	require.True(t, ok, "fetch_orders should be a map")
	assert.Equal(t, 456, fetchOrdersData["id"])
	assert.Equal(t, 99.99, fetchOrdersData["total"])

	fetchSettingsData, ok := vars["fetch_settings"].(map[string]any)
	require.True(t, ok, "fetch_settings should be a map")
	assert.Equal(t, "dark", fetchSettingsData["theme"])

	// Global context should be accessible
	globalVal, ok := inputs.GlobalContext.Get("config_value")
	assert.True(t, ok)
	assert.Equal(t, "global_config", globalVal)

	// Parent outputs should be available
	assert.Len(t, inputs.ParentOutputs, 3)
}

// Helper functions for tests

type testNode struct {
	id            uuid.UUID
	name          string
	bindingConfig *domain.InputBindingConfig
	schema        *domain.NodeIOSchema
}

func (n *testNode) ID() uuid.UUID                  { return n.id }
func (n *testNode) Type() domain.NodeType          { return domain.NodeTypeTransform }
func (n *testNode) Name() string                   { return n.name }
func (n *testNode) Config() map[string]any         { return make(map[string]any) }
func (n *testNode) IOSchema() *domain.NodeIOSchema { return n.schema }
func (n *testNode) InputBindingConfig() *domain.InputBindingConfig {
	if n.bindingConfig == nil {
		return &domain.InputBindingConfig{
			AutoBind:          true,
			Mappings:          make(map[string]string),
			CollisionStrategy: domain.CollisionStrategyNamespaceByParent,
		}
	}
	return n.bindingConfig
}

func createTestNode(name string, bindingConfig *domain.InputBindingConfig) domain.Node {
	return &testNode{
		id:            uuid.New(),
		name:          name,
		bindingConfig: bindingConfig,
	}
}

type testExecution struct {
	id              uuid.UUID
	workflowID      uuid.UUID
	triggerID       uuid.UUID
	globalVariables *domain.VariableSet
	nodeOutputs     map[uuid.UUID]*domain.VariableSet
	variables       *domain.VariableSet
}

func (e *testExecution) ID() uuid.UUID                        { return e.id }
func (e *testExecution) WorkflowID() uuid.UUID                { return e.workflowID }
func (e *testExecution) TriggerID() uuid.UUID                 { return e.triggerID }
func (e *testExecution) GlobalVariables() *domain.VariableSet { return e.globalVariables }
func (e *testExecution) GetNodeOutput(nodeID uuid.UUID) (*domain.VariableSet, bool) {
	output, ok := e.nodeOutputs[nodeID]
	return output, ok
}
func (e *testExecution) SetNodeOutput(nodeID uuid.UUID, output map[string]any) error {
	vars := domain.NewVariableSet(nil)
	for k, v := range output {
		if err := vars.Set(k, v); err != nil {
			return err
		}
	}
	e.nodeOutputs[nodeID] = vars
	return nil
}

// Unused methods to satisfy interface
func (e *testExecution) Phase() domain.ExecutionPhase { return domain.ExecutionPhaseExecuting }
func (e *testExecution) StartedAt() time.Time         { return time.Now() }
func (e *testExecution) FinishedAt() *time.Time       { return nil }
func (e *testExecution) Duration() time.Duration      { return 0 }
func (e *testExecution) Variables() *domain.VariableSet {
	if e.variables == nil {
		e.variables = domain.NewVariableSet(nil)
	}
	return e.variables
}
func (e *testExecution) GetVariable(key string) (any, bool) {
	return e.Variables().Get(key)
}
func (e *testExecution) SetGlobalVariable(key string, value any) error {
	return e.globalVariables.Set(key, value)
}
func (e *testExecution) GetNodeState(uuid.UUID) (*domain.NodeExecutionState, bool)  { return nil, false }
func (e *testExecution) GetAllNodeStates() map[uuid.UUID]*domain.NodeExecutionState { return nil }
func (e *testExecution) Error() string                                              { return "" }
func (e *testExecution) HasError() bool                                             { return false }
func (e *testExecution) GetUncommittedEvents() []domain.Event                       { return nil }
func (e *testExecution) MarkEventsAsCommitted()                                     {}
func (e *testExecution) ApplyEvent(event domain.Event) error                        { return nil }

// Command methods
func (e *testExecution) Start(uuid.UUID, map[string]any) error { return nil }
func (e *testExecution) StartNode(uuid.UUID, string, domain.NodeType, map[string]any) error {
	return nil
}
func (e *testExecution) CompleteNode(uuid.UUID, string, domain.NodeType, map[string]any, time.Duration) error {
	return nil
}
func (e *testExecution) FailNode(uuid.UUID, string, domain.NodeType, string, int) error { return nil }
func (e *testExecution) SkipNode(uuid.UUID, string, string) error                       { return nil }
func (e *testExecution) SetVariable(string, any, domain.VariableScope, uuid.UUID) error { return nil }
func (e *testExecution) Complete(map[string]any) error                                  { return nil }
func (e *testExecution) Fail(string, uuid.UUID) error                                   { return nil }

func createTestExecution(globalVars map[string]any) *testExecution {
	globals := domain.NewVariableSet(nil)
	for k, v := range globalVars {
		globals.Set(k, v)
	}
	globals.SetReadOnly(true)

	return &testExecution{
		id:              uuid.New(),
		workflowID:      uuid.New(),
		triggerID:       uuid.New(),
		globalVariables: globals,
		nodeOutputs:     make(map[uuid.UUID]*domain.VariableSet),
	}
}

type testEdge struct {
	sourceID uuid.UUID
	targetID uuid.UUID
}

func createTestGraph(nodes []domain.Node, edges []testEdge) *WorkflowGraph {
	nodeMap := make(map[uuid.UUID]domain.Node)
	nodeList := make([]domain.Node, 0, len(nodes))
	for _, node := range nodes {
		nodeMap[node.ID()] = node
		nodeList = append(nodeList, node)
	}

	edgeList := make([]domain.Edge, 0, len(edges))
	forwardEdges := make(map[uuid.UUID][]domain.Edge)
	reverseEdges := make(map[uuid.UUID][]domain.Edge)

	// Create edges
	for _, e := range edges {
		edge := &testGraphEdge{
			id:       uuid.New(),
			sourceID: e.sourceID,
			targetID: e.targetID,
		}
		edgeList = append(edgeList, edge)
		reverseEdges[e.targetID] = append(reverseEdges[e.targetID], edge)
		forwardEdges[e.sourceID] = append(forwardEdges[e.sourceID], edge)
	}

	graph := &WorkflowGraph{
		workflowID:   uuid.New(),
		nodes:        nodeMap,
		nodeList:     nodeList,
		edges:        edgeList,
		forwardEdges: forwardEdges,
		reverseEdges: reverseEdges,
	}

	return graph
}

type testGraphEdge struct {
	id       uuid.UUID
	sourceID uuid.UUID
	targetID uuid.UUID
}

func (e *testGraphEdge) ID() uuid.UUID          { return e.id }
func (e *testGraphEdge) FromNodeID() uuid.UUID  { return e.sourceID }
func (e *testGraphEdge) ToNodeID() uuid.UUID    { return e.targetID }
func (e *testGraphEdge) Type() domain.EdgeType  { return domain.EdgeTypeDirect }
func (e *testGraphEdge) Config() map[string]any { return make(map[string]any) }
