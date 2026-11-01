package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func rcImporter(types ...string) *YAMLImporter {
	m := executor.NewManager()
	for _, typ := range types {
		_ = m.Register(typ, &executor.ExecutorFunc{})
	}
	return NewYAMLImporter(m)
}

const rcNodes = `
metadata:
  name: wf
nodes:
  - {id: a, name: A, type: http}
  - {id: b, name: B, type: http}
`

func TestRCImportFromYAML_MalformedYAML(t *testing.T) {
	_, err := rcImporter("http").ImportFromYAML([]byte("metadata: [unclosed"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestRCImportFromYAML_EdgeValidation(t *testing.T) {
	cases := map[string]struct {
		edges string
		field string
		msg   string
	}{
		"missing id":     {"  - {from: a, to: b}\n", "edges[0].id", "edge ID is required"},
		"duplicate id":   {"  - {id: e, from: a, to: b}\n  - {id: e, from: b, to: a}\n", "edges[1].id", "duplicate edge ID: e"},
		"missing source": {"  - {id: e, to: b}\n", "edges[0].from", "edge source is required"},
		"missing target": {"  - {id: e, from: a}\n", "edges[0].to", "edge target is required"},
		"unknown source": {"  - {id: e, from: zz, to: b}\n", "edges[0].from", "non-existent source node: zz"},
		"unknown target": {"  - {id: e, from: a, to: zz}\n", "edges[0].to", "non-existent target node: zz"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := rcImporter("http").ImportFromYAML([]byte(rcNodes + "edges:\n" + tc.edges))
			require.Error(t, err)
			var vErr *ValidationError
			require.ErrorAs(t, err, &vErr)
			assert.Equal(t, tc.field, vErr.Field)
			assert.Contains(t, vErr.Message, tc.msg)
		})
	}
}

func TestRCImportFromYAML_TriggerValidation(t *testing.T) {
	_, err := rcImporter("http").ImportFromYAML([]byte(rcNodes + "trigger:\n  type: manual\n"))
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "trigger.name", vErr.Field)

	_, err = rcImporter("http").ImportFromYAML([]byte(rcNodes + "trigger:\n  name: t\n"))
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "trigger.type", vErr.Field)
}

func TestRCImportFromYAML_TriggerDefaults(t *testing.T) {
	res, err := rcImporter("http").ImportFromYAML([]byte(rcNodes + "trigger:\n  name: t\n  type: manual\n"))
	require.NoError(t, err)
	require.NotNil(t, res.Trigger)
	assert.True(t, res.Trigger.Enabled, "triggers are enabled by default")
	assert.NotNil(t, res.Trigger.Config, "a nil config becomes an empty map")
	assert.Equal(t, res.Workflow.ID, res.Trigger.WorkflowID)

	res, err = rcImporter("http").ImportFromYAML([]byte(rcNodes + "trigger:\n  name: t\n  type: manual\n  enabled: false\n"))
	require.NoError(t, err)
	assert.False(t, res.Trigger.Enabled)
}

func TestRCImportFromYAML_TriggerDomainValidation(t *testing.T) {
	// A cron trigger without a schedule passes the YAML checks but fails
	// domain validation.
	_, err := rcImporter("http").ImportFromYAML([]byte(rcNodes + "trigger:\n  name: t\n  type: cron\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trigger validation failed")
	assert.Contains(t, err.Error(), "cron schedule is required")

	res, err := rcImporter("http").ImportFromYAML([]byte(rcNodes + "trigger:\n  name: t\n  type: cron\n  config: {schedule: \"*/5 * * * *\"}\n"))
	require.NoError(t, err)
	assert.Equal(t, "*/5 * * * *", res.Trigger.Config["schedule"])
}

func TestRCImportFromYAML_CyclesAreAllowed(t *testing.T) {
	cyclic := rcNodes + "edges:\n  - {id: e1, from: a, to: b}\n  - {id: e2, from: b, to: a}\n"
	res, err := rcImporter("http").ImportFromYAML([]byte(cyclic))
	require.NoError(t, err)
	assert.Equal(t, 2, res.EdgesCount)
}

func TestRCConvertToTrigger_Nil(t *testing.T) {
	assert.Nil(t, rcImporter().convertToTrigger(&YAMLWorkflow{}, "wf"))
}

func TestRCNodeTypeHelpers(t *testing.T) {
	withoutManager := NewYAMLImporter(nil)
	assert.Nil(t, withoutManager.GetSupportedNodeTypes())
	assert.True(t, withoutManager.ValidateNodeType("anything"), "all types are allowed without a registry")

	imp := rcImporter("http", "llm")
	assert.ElementsMatch(t, []string{"http", "llm"}, imp.GetSupportedNodeTypes())
	assert.True(t, imp.ValidateNodeType("llm"))
	assert.False(t, imp.ValidateNodeType("nope"))

	// Without a registry, unknown node types are accepted on import.
	res, err := withoutManager.ImportFromYAML([]byte("metadata: {name: x}\nnodes:\n  - {id: a, name: A, type: custom}\n"))
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowStatus("draft"), res.Workflow.Status)
}
