package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
)

func TestTriggers_NameAndDescriptionPersisted(t *testing.T) {
	ctx := context.Background()
	env := rcNewEnv()
	wf := rcWorkflowModel("wf", "active")
	env.workflows.put(wf)

	created, err := env.srv.CreateTrigger(ctx, &pb.CreateTriggerRequest{
		WorkflowId: wf.ID.String(), Name: "nightly", Description: "runs at night", Type: "manual", Enabled: true,
	})
	require.NoError(t, err)
	id := created.Trigger.Id

	got, err := env.srv.GetTrigger(ctx, &pb.GetTriggerRequest{Id: id})
	require.NoError(t, err)
	assert.Equal(t, "nightly", got.Trigger.Name)
	assert.Equal(t, "runs at night", got.Trigger.Description)

	// An empty proto description leaves the stored one untouched.
	upd, err := env.srv.UpdateTrigger(ctx, &pb.UpdateTriggerRequest{Id: id, Name: "renamed"})
	require.NoError(t, err)
	assert.Equal(t, "renamed", upd.Trigger.Name)
	assert.Equal(t, "runs at night", upd.Trigger.Description)

	upd, err = env.srv.UpdateTrigger(ctx, &pb.UpdateTriggerRequest{Id: id, Description: "new"})
	require.NoError(t, err)
	assert.Equal(t, "renamed", upd.Trigger.Name)
	assert.Equal(t, "new", upd.Trigger.Description)

	list, err := env.srv.ListTriggers(ctx, &pb.ListTriggersRequest{})
	require.NoError(t, err)
	require.Len(t, list.Triggers, 1)
	assert.Equal(t, "renamed", list.Triggers[0].Name)
	assert.Equal(t, "new", list.Triggers[0].Description)
}
