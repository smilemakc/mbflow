package storage

import (
	"context"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore_WorkflowsAndExecutions(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	w := NewWorkflowBuilder().ID("w1").Name("demo").Version("1").Build()
	err := s.SaveWorkflow(ctx, w)
	assert.NoError(t, err)

	got, err := s.GetWorkflow(ctx, "w1")
	assert.NoError(t, err)
	assert.Equal(t, "demo", got.Name())

	x := NewExecutionBuilder().ID("e1").WorkflowID("w1").Status(domain.ExecutionStatusRunning).Build()
	err = s.SaveExecution(ctx, x)
	assert.NoError(t, err)

	xgot, err := s.GetExecution(ctx, "e1")
	assert.NoError(t, err)
	assert.Equal(t, "w1", xgot.WorkflowID())

	ev := NewEventBuilder().EventID("ev1").EventType("WorkflowStarted").WorkflowID("w1").ExecutionID("e1").Timestamp(time.Now()).Build()
	err = s.AppendEvent(ctx, ev)
	assert.NoError(t, err)

	evs, err := s.ListEventsByExecution(ctx, "e1")
	assert.NoError(t, err)
	assert.NotEmpty(t, evs)
}
