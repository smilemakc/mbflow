package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore_WorkflowsAndExecutions(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	w, err := NewWorkflowBuilder().
		ID(uuid.New()).
		Name("demo").
		Version("1").
		Description("test workflow").
		Build()
	assert.NoError(t, err)
	err = s.SaveWorkflow(ctx, w)
	assert.NoError(t, err)

	got, err := s.GetWorkflow(ctx, w.ID())
	assert.NoError(t, err)
	assert.Equal(t, "demo", got.Name())

	x, err := NewExecutionBuilder().ID(uuid.New()).WorkflowID(w.ID()).Build()
	assert.NoError(t, err)
	err = s.SaveExecution(ctx, x)
	assert.NoError(t, err)

	xgot, err := s.GetExecution(ctx, x.ID())
	assert.NoError(t, err)
	assert.Equal(t, w.ID(), xgot.WorkflowID())

	ev := NewEventBuilder().
		EventID(uuid.New()).
		EventType(domain.EventTypeExecutionStarted).
		WorkflowID(w.ID()).
		ExecutionID(x.ID()).
		Timestamp(time.Now()).
		SequenceNumber(1).
		Build()
	err = s.AppendEvent(ctx, ev)
	assert.NoError(t, err)

	evs, err := s.ListEventsByExecution(ctx, x.ID())
	assert.NoError(t, err)
	assert.NotEmpty(t, evs)
}
