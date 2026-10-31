package builder

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func rcFailingWorkflowOption() WorkflowOption {
	return func(*WorkflowBuilder) error { return errors.New("option failed") }
}

func rcFailingNodeOption() NodeOption {
	return func(*NodeBuilder) error { return errors.New("node option failed") }
}

func TestRCEdgeOptions(t *testing.T) {
	e, err := NewEdge("cond", "yes", FromTrueBranch()).Build()
	require.NoError(t, err)
	assert.Equal(t, "true", e.SourceHandle)

	e, err = NewEdge("cond", "no", FromFalseBranch()).Build()
	require.NoError(t, err)
	assert.Equal(t, "false", e.SourceHandle)

	e, err = NewEdge("a", "b", WithSourceHandle("out")).Build()
	require.NoError(t, err)
	assert.Equal(t, "out", e.SourceHandle)

	_, err = NewEdge("a", "b", WithSourceHandle("")).Build()
	assert.ErrorContains(t, err, "source handle cannot be empty")

	e, err = NewEdge("b", "a", WithLoop(5)).Build()
	require.NoError(t, err)
	require.NotNil(t, e.Loop)
	assert.Equal(t, 5, e.Loop.MaxIterations)
	assert.True(t, e.IsLoop())

	_, err = NewEdge("b", "a", WithLoop(0)).Build()
	assert.ErrorContains(t, err, "max iterations must be > 0")

	// Edge-level validation errors surface from Build.
	_, err = NewEdge("a", "a").Build()
	assert.Error(t, err)
}

func TestRCConditionalLoopWorkflow(t *testing.T) {
	wf, err := NewWorkflow("loop").
		AddNode(NewPassthroughNode("start", "Start")).
		AddNode(NewNode("check", "conditional", "Check", WithConfigValue("condition", "x > 1"))).
		AddNode(NewPassthroughNode("done", "Done")).
		Connect("start", "check").
		Connect("check", "done", FromTrueBranch()).
		Connect("check", "start", FromFalseBranch(), WithLoop(3)).
		Build()
	require.NoError(t, err)
	require.Len(t, wf.Edges, 3)
	assert.Equal(t, "false", wf.Edges[2].SourceHandle)
	assert.Equal(t, 3, wf.Edges[2].Loop.MaxIterations)
}

func TestRCNodeHelpers(t *testing.T) {
	n, err := NewSubWorkflowNode("sub", "Sub", "wf-1", WithForEach("{{input.items}}"), WithWorkflowID("wf-2")).Build()
	require.NoError(t, err)
	assert.Equal(t, "sub_workflow", n.Type)
	assert.Equal(t, "wf-2", n.Config["workflow_id"])

	_, err = NewSubWorkflowNode("sub", "Sub", "wf-1", rcFailingNodeOption()).Build()
	assert.ErrorContains(t, err, "node option failed")

	n, err = NewGeminiNode("g", "Gemini", "gemini-2.0", "hi").Build()
	require.NoError(t, err)
	assert.Equal(t, string(models.LLMProviderGemini), n.Config["provider"])
	assert.Equal(t, "gemini-2.0", n.Config["model"])

	// Node-level validation errors surface from Build.
	_, err = NewNode("", "http", "x").Build()
	assert.Error(t, err)
}

func TestRCPositions(t *testing.T) {
	n, err := NewPassthroughNode("b", "B", RelativePosition("a", 50, 75)).Build()
	require.NoError(t, err)
	require.NotNil(t, n.Position)
	assert.Equal(t, models.Position{X: 50, Y: 75}, *n.Position)
	assert.Equal(t, map[string]any{"ref_node": "a", "offset_x": 50.0, "offset_y": 75.0}, n.Metadata["_position_ref"])

	nb := &NodeBuilder{id: "x", nodeType: "t", name: "X", config: map[string]any{}}
	require.NoError(t, RelativePosition("a", 1, 2)(nb))
	assert.NotNil(t, nb.metadata)

	_, err = NewPassthroughNode("b", "B", RelativePosition("", 0, 0)).Build()
	assert.ErrorContains(t, err, "reference node ID cannot be empty")

	// AutoLayoutPosition leaves position to the workflow builder.
	wf, err := NewWorkflow("auto", WithAutoLayout()).
		AddNode(NewPassthroughNode("a", "A", AutoLayoutPosition())).
		AddNode(NewPassthroughNode("b", "B", AutoLayoutPosition())).
		Connect("a", "b").
		Build()
	require.NoError(t, err)
	assert.Equal(t, 0.0, wf.Nodes[0].Position.X)
	assert.Equal(t, 200.0, wf.Nodes[1].Position.X)
}

func TestRCWorkflowOptionErrors(t *testing.T) {
	_, err := NewWorkflow("w", rcFailingWorkflowOption()).Build()
	assert.ErrorContains(t, err, "option failed")

	_, err = NewWorkflow("w", WithVariable("", 1)).Build()
	assert.ErrorContains(t, err, "variable key cannot be empty")

	_, err = NewWorkflow("w", WithMetadata("", 1)).Build()
	assert.ErrorContains(t, err, "metadata key cannot be empty")

	// Errors stop further chaining.
	wb := NewWorkflow("w", WithVariable("", 1))
	assert.Same(t, wb, wb.AddNode(NewPassthroughNode("a", "A")).Connect("a", "b"))
	assert.Empty(t, wb.nodes)
	assert.Empty(t, wb.edges)

	_, err = NewWorkflow("w").AddNode(nil).Build()
	assert.ErrorContains(t, err, "node builder cannot be nil")

	_, err = NewWorkflow("w").AddNode(NewPassthroughNode("", "A")).Build()
	assert.ErrorContains(t, err, "node must have an ID")

	_, err = NewWorkflow("w").AddNode(NewPassthroughNode("a", "")).Build()
	assert.ErrorContains(t, err, "node a")

	_, err = NewWorkflow("w").
		AddNode(NewPassthroughNode("a", "A")).
		AddNode(NewPassthroughNode("b", "B")).
		Connect("a", "b", WithLoop(-1)).
		Build()
	assert.ErrorContains(t, err, "edge 0")

	_, err = NewWorkflow("").AddNode(NewPassthroughNode("a", "A")).Build()
	assert.ErrorContains(t, err, "validation failed")
}

func TestRCStrictValidation(t *testing.T) {
	// Without strict mode an incomplete HTTP config is accepted.
	_, err := NewWorkflow("lenient").
		AddNode(NewNode("h", "http", "H", HTTPURL("https://example.com"))).
		Build()
	require.NoError(t, err)

	_, err = NewWorkflow("strict", WithStrictValidation()).
		AddNode(NewNode("h", "http", "H", HTTPURL("https://example.com"))).
		Build()
	assert.ErrorContains(t, err, "node h: HTTP node requires 'method' field")

	_, err = NewWorkflow("strict", WithStrictValidation()).
		AddNode(NewOpenAINode("l", "L", "gpt-4o", "hi")).
		Build()
	assert.ErrorContains(t, err, "api_key")

	wf, err := NewWorkflow("strict", WithStrictValidation()).
		AddNode(NewHTTPGetNode("h", "H", "https://example.com")).
		AddNode(NewOpenAINode("l", "L", "gpt-4o", "hi", LLMAPIKey("k"))).
		AddNode(NewNode("c", "custom", "C")).
		Connect("h", "l").
		Connect("l", "c").
		Build()
	require.NoError(t, err)
	assert.Len(t, wf.Nodes, 3)
}
