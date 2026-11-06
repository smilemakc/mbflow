package rest

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/testutil"
)

func TestEdgeCondition_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		in   string
		want EdgeCondition
	}{
		"string":       {`"output.ok == true"`, EdgeCondition{"expression": "output.ok == true"}},
		"object":       {`{"expression":"output.ok == true"}`, EdgeCondition{"expression": "output.ok == true"}},
		"empty string": {`""`, nil},
	} {
		var got EdgeCondition
		require.NoError(t, json.Unmarshal([]byte(tc.in), &got), name)
		assert.Equal(t, tc.want, got, name)
	}

	var got EdgeCondition
	require.ErrorContains(t, json.Unmarshal([]byte(`42`), &got), "expression string")
}

// A workflow read with GET (conditions as strings) can be sent back with PUT,
// and conditions sent as strings are stored.
func TestHandlers_Workflow_EdgeConditionRoundTrip(t *testing.T) {
	t.Parallel()
	_, router, cleanup := setupWorkflowHandlersTest(t)
	defer cleanup()

	node := func(id string) map[string]any {
		return map[string]any{"id": id, "name": id, "type": "transform", "config": map[string]any{"type": "template", "template": "x"}}
	}
	w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", map[string]any{
		"name":  "Conditions",
		"nodes": []any{node("a"), node("b"), node("c")},
		"edges": []any{
			map[string]any{"id": "e1", "from": "a", "to": "b", "condition": `output.value == "x"`},
			map[string]any{"id": "e2", "from": "a", "to": "c", "condition": map[string]any{"expression": `output.value != "x"`}},
		},
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created map[string]any
	testutil.ParseResponse(t, w, &created)
	id := created["id"].(string)

	w = testutil.MakeRequest(t, router, "GET", "/api/v1/workflows/"+id, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var fetched map[string]any
	testutil.ParseResponse(t, w, &fetched)
	conditions := map[string]any{}
	for _, e := range fetched["edges"].([]any) {
		edge := e.(map[string]any)
		conditions[edge["id"].(string)] = edge["condition"]
	}
	assert.Equal(t, map[string]any{"e1": `output.value == "x"`, "e2": `output.value != "x"`}, conditions)

	w = testutil.MakeRequest(t, router, "PUT", "/api/v1/workflows/"+id, map[string]any{
		"name": fetched["name"], "nodes": fetched["nodes"], "edges": fetched["edges"],
	})
	require.Equal(t, http.StatusOK, w.Code, "GET output is accepted by PUT: %s", w.Body.String())
}
