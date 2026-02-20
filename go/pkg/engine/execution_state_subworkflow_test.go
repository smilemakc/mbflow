package engine

import (
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func TestExecutionState_ParentFields(t *testing.T) {
	t.Parallel()

	wf := &models.Workflow{ID: "wf-1", Name: "Test"}
	state := NewExecutionState("exec-1", "wf-1", wf, nil, nil)

	if state.ParentExecutionID != "" {
		t.Fatal("expected empty parent execution ID")
	}
	if state.ItemIndex != nil {
		t.Fatal("expected nil item index")
	}

	state.ParentExecutionID = "parent-exec-1"
	state.ParentNodeID = "fanout-node"
	idx := 3
	state.ItemIndex = &idx
	state.ItemKey = "cell_3"

	if state.ParentExecutionID != "parent-exec-1" {
		t.Fatal("parent execution ID mismatch")
	}
	if *state.ItemIndex != 3 {
		t.Fatal("item index mismatch")
	}
	if state.ItemKey != "cell_3" {
		t.Fatal("item key mismatch")
	}
}
