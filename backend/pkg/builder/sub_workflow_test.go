package builder

import (
	"testing"
)

func TestNewSubWorkflowNode(t *testing.T) {
	t.Parallel()

	nb := NewSubWorkflowNode("fanout", "Generate Cells", "child-wf-id",
		WithForEach("input.cells"),
		WithItemVar("cell"),
		WithMaxParallelism(5),
		WithOnError("collect_partial"),
	)

	if nb.id != "fanout" {
		t.Fatalf("expected id=fanout, got: %s", nb.id)
	}
	if nb.nodeType != "sub_workflow" {
		t.Fatalf("expected type=sub_workflow, got: %s", nb.nodeType)
	}
	if nb.config["workflow_id"] != "child-wf-id" {
		t.Fatalf("expected workflow_id=child-wf-id, got: %v", nb.config["workflow_id"])
	}
	if nb.config["for_each"] != "input.cells" {
		t.Fatalf("expected for_each=input.cells, got: %v", nb.config["for_each"])
	}
	if nb.config["item_var"] != "cell" {
		t.Fatalf("expected item_var=cell, got: %v", nb.config["item_var"])
	}
	if nb.config["max_parallelism"] != 5 {
		t.Fatalf("expected max_parallelism=5, got: %v", nb.config["max_parallelism"])
	}
	if nb.config["on_error"] != "collect_partial" {
		t.Fatalf("expected on_error=collect_partial, got: %v", nb.config["on_error"])
	}
}

func TestSubWorkflowNode_InWorkflow(t *testing.T) {
	t.Parallel()

	wf, err := NewWorkflow("Test WF").
		AddNode(NewNode("source", "transform", "Source")).
		AddNode(NewSubWorkflowNode("fanout", "Fan Out", "child-wf",
			WithForEach("input.items"),
		)).
		AddNode(NewNode("sink", "transform", "Sink")).
		Connect("source", "fanout").
		Connect("fanout", "sink").
		Build()

	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	if len(wf.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got: %d", len(wf.Nodes))
	}

	var found bool
	for _, n := range wf.Nodes {
		if n.ID == "fanout" {
			found = true
			if n.Type != "sub_workflow" {
				t.Fatalf("expected type=sub_workflow, got: %s", n.Type)
			}
			break
		}
	}
	if !found {
		t.Fatal("fanout node not found")
	}
}
