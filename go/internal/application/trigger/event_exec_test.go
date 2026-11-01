package trigger

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// newReplicaListener creates an event listener as one API replica would,
// sharing the given Redis with the other replicas.
func newReplicaListener(t *testing.T, rc *cache.RedisCache, exec WorkflowExecutor) (*EventListener, *mockTriggerRepo) {
	t.Helper()
	repo := &mockTriggerRepo{}
	repo.On("MarkTriggered", mock.Anything, mock.Anything).Return(nil).Maybe()
	el, err := NewEventListener(EventListenerConfig{
		TriggerRepo:  repo,
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: exec,
		Cache:        rc,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = el.Stop() })
	return el, repo
}

func eventTriggerModel(eventType string) *storagemodels.TriggerModel {
	return &storagemodels.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Type:       string(models.TriggerTypeEvent),
		Enabled:    true,
		Config: storagemodels.JSONBMap{
			"event_type": eventType,
			"input":      map[string]any{"source": "default", "keep": true},
		},
	}
}

func waitSubscribed(t *testing.T, rc *cache.RedisCache, channel string, n int64) {
	t.Helper()
	require.Eventually(t, func() bool {
		res, err := rc.Client().PubSubNumSub(context.Background(), channel).Result()
		return err == nil && res[channel] >= n
	}, 2*time.Second, 5*time.Millisecond)
}

func TestEventListener_TwoReplicasFireOncePerEvent(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	exec := &fakeExecutor{}
	tm := eventTriggerModel("order.paid")
	ctx := context.Background()

	replicaA, _ := newReplicaListener(t, rc, exec)
	replicaB, _ := newReplicaListener(t, rc, exec)
	require.NoError(t, replicaA.Start(ctx, []*storagemodels.TriggerModel{tm}))
	require.NoError(t, replicaB.Start(ctx, []*storagemodels.TriggerModel{tm}))
	waitSubscribed(t, rc, "mbflow:events:order.paid", 2)

	require.NoError(t, PublishEvent(ctx, rc, Event{Type: "order.paid", Source: "billing", Data: map[string]any{"source": "event", "amount": 10.0}}))
	require.NoError(t, PublishEvent(ctx, rc, Event{Type: "order.paid", Source: "billing", Data: map[string]any{"amount": 20.0}}))

	require.Eventually(t, func() bool { return exec.Count() == 2 }, 2*time.Second, 5*time.Millisecond)
	// Both replicas received both events; give the losing deliveries time
	// to (not) execute.
	assert.Never(t, func() bool { return exec.Count() > 2 }, 150*time.Millisecond, 10*time.Millisecond)

	claims := 0
	for _, k := range mr.Keys() {
		if len(k) > len(claimKeyPrefix) && k[:len(claimKeyPrefix)] == claimKeyPrefix {
			claims++
		}
	}
	assert.Equal(t, 2, claims)

	// Event data overrides the trigger's default input.
	for _, c := range exec.Calls() {
		assert.Equal(t, tm.ID.String(), c.TriggerID)
		assert.Equal(t, true, c.Input["keep"])
		if c.Input["amount"] == 10.0 {
			assert.Equal(t, "event", c.Input["source"])
		} else {
			assert.Equal(t, "default", c.Input["source"])
		}
	}
}

func TestEventListener_Fire_DedupAndFailOpen(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	exec := &fakeExecutor{}
	a, repoA := newReplicaListener(t, rc, exec)
	b, _ := newReplicaListener(t, rc, exec)
	trig := a.modelToDomain(eventTriggerModel("x"))
	ev := Event{ID: "evt-1", Type: "x", Data: map[string]any{"k": "v"}}

	assert.True(t, a.fire(trig, ev, ev.ID))
	assert.False(t, b.fire(trig, ev, ev.ID), "same event, same trigger: second replica must skip")
	assert.Equal(t, 1, exec.Count())
	repoA.AssertNumberOfCalls(t, "MarkTriggered", 1)

	state, err := LoadTriggerState(context.Background(), rc, trig.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), state.ExecutionCount)

	// Another trigger on the same event is an independent occurrence.
	other := a.modelToDomain(eventTriggerModel("x"))
	assert.True(t, b.fire(other, ev, ev.ID))

	// Redis errors fail open.
	mr.SetError("ERR redis unavailable")
	assert.True(t, a.fire(trig, Event{ID: "evt-2", Type: "x"}, "evt-2"))
	assert.Equal(t, 3, exec.Count())
}

func TestEventListener_ExecuteTrigger_Errors(t *testing.T) {
	rc := setupTestRedis(t)
	exec := &fakeExecutor{err: errExec}
	el, repo := newReplicaListener(t, rc, exec)
	trig := el.modelToDomain(eventTriggerModel("x"))

	err := el.executeTrigger(context.Background(), trig, nil)
	require.ErrorIs(t, err, errExec)
	repo.AssertNotCalled(t, "MarkTriggered", mock.Anything, mock.Anything)

	// fire logs the error and still reports the occurrence as handled.
	assert.True(t, el.fire(trig, Event{ID: "e"}, "e"))

	// MarkTriggered failures are logged only.
	exec.err = nil
	repo.ExpectedCalls = nil
	repo.On("MarkTriggered", mock.Anything, mock.Anything).Return(errExec).Once()
	require.NoError(t, el.executeTrigger(context.Background(), trig, map[string]any{"a": 1}))
	repo.AssertExpectations(t)
}

func TestEventListener_FirstTriggerAfterStartSubscribes(t *testing.T) {
	rc := setupTestRedis(t)
	exec := &fakeExecutor{}
	el, _ := newReplicaListener(t, rc, exec)
	ctx := context.Background()

	// Started with no event triggers (e.g. none existed at boot).
	require.NoError(t, el.Start(ctx, nil))

	// A trigger created later through the API must start receiving events
	// without a restart.
	trig := el.modelToDomain(eventTriggerModel("user.created"))
	reqCtx, cancel := context.WithCancel(ctx)
	require.NoError(t, el.AddTrigger(reqCtx, trig))
	cancel() // the request context ending must not stop the listener
	waitSubscribed(t, rc, "mbflow:events:user.created", 1)

	// A second event type on the open subscription.
	trig2 := el.modelToDomain(eventTriggerModel("user.deleted"))
	require.NoError(t, el.AddTrigger(ctx, trig2))
	waitSubscribed(t, rc, "mbflow:events:user.deleted", 1)

	require.NoError(t, PublishEvent(ctx, rc, Event{Type: "user.created"}))
	require.Eventually(t, func() bool { return exec.Count() == 1 }, 2*time.Second, 5*time.Millisecond)

	// Removing the last trigger of a type unsubscribes from it.
	require.NoError(t, el.RemoveTrigger(ctx, trig2.ID))
	require.Eventually(t, func() bool {
		res, err := rc.Client().PubSubNumSub(ctx, "mbflow:events:user.deleted").Result()
		return err == nil && res["mbflow:events:user.deleted"] == 0
	}, 2*time.Second, 5*time.Millisecond)

	require.NoError(t, el.Stop())
	require.NoError(t, el.Stop(), "Stop is idempotent")

	// After Stop new triggers are recorded but no subscription is opened.
	el2, _ := newReplicaListener(t, rc, exec)
	require.NoError(t, el2.Start(ctx, nil))
	require.NoError(t, el2.Stop())
	require.NoError(t, el2.AddTrigger(ctx, trig))
	assert.Nil(t, el2.pubsub)
}

func TestEventListener_IgnoresBadPayloadsAndNonMatchingEvents(t *testing.T) {
	rc := setupTestRedis(t)
	exec := &fakeExecutor{}
	el, _ := newReplicaListener(t, rc, exec)
	ctx := context.Background()

	tm := eventTriggerModel("alert")
	tm.Config["filter"] = map[string]any{"source": "monitor"}
	require.NoError(t, el.Start(ctx, []*storagemodels.TriggerModel{tm}))
	waitSubscribed(t, rc, "mbflow:events:alert", 1)

	require.NoError(t, rc.Client().Publish(ctx, "mbflow:events:alert", "not json").Err())
	require.NoError(t, PublishEvent(ctx, rc, Event{Type: "alert", Source: "other"}))
	// Published without an ID by a foreign publisher: still handled once.
	raw, err := json.Marshal(Event{Type: "alert", Source: "monitor"})
	require.NoError(t, err)
	require.NoError(t, rc.Client().Publish(ctx, "mbflow:events:alert", string(raw)).Err())

	require.Eventually(t, func() bool { return exec.Count() == 1 }, 2*time.Second, 5*time.Millisecond)
	assert.Never(t, func() bool { return exec.Count() > 1 }, 100*time.Millisecond, 10*time.Millisecond)
}

func TestEventListener_ListenStopsWhenPubSubCloses(t *testing.T) {
	rc := setupTestRedis(t)
	el, _ := newReplicaListener(t, rc, &fakeExecutor{})
	require.NoError(t, el.Start(context.Background(), []*storagemodels.TriggerModel{eventTriggerModel("a")}))

	// Closing the connection underneath the listener ends the goroutine.
	require.NoError(t, el.pubsub.Close())
	select {
	case <-el.stoppedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not stop after pub/sub closed")
	}

	// Stop still completes (closing an already closed pub/sub is reported).
	done := make(chan error, 1)
	go func() { done <- el.Stop() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop hung")
	}
}

func TestEventListener_StartSkipsInvalidTriggers(t *testing.T) {
	rc := setupTestRedis(t)
	el, _ := newReplicaListener(t, rc, &fakeExecutor{})
	bad := eventTriggerModel("")
	require.NoError(t, el.Start(context.Background(), []*storagemodels.TriggerModel{bad}))
	assert.Empty(t, el.triggers)
	assert.Nil(t, el.pubsub)
}

func TestPublishEvent_AssignsIDAndTimestamp(t *testing.T) {
	rc := setupTestRedis(t)
	ctx := context.Background()
	sub := rc.Client().Subscribe(ctx, "mbflow:events:ping")
	defer sub.Close()
	_, err := sub.Receive(ctx) // subscription confirmation
	require.NoError(t, err)

	require.NoError(t, PublishEvent(ctx, rc, Event{Type: "ping"}))
	require.NoError(t, PublishEvent(ctx, rc, Event{ID: "fixed", Type: "ping"}))

	var got []Event
	for i := 0; i < 2; i++ {
		msg, err := sub.ReceiveMessage(ctx)
		require.NoError(t, err)
		var ev Event
		require.NoError(t, json.Unmarshal([]byte(msg.Payload), &ev))
		got = append(got, ev)
	}
	_, err = uuid.Parse(got[0].ID)
	assert.NoError(t, err, "PublishEvent assigns a UUID")
	assert.False(t, got[0].Timestamp.IsZero())
	assert.Equal(t, "fixed", got[1].ID)
}

func TestPublishEvent_Errors(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	ctx := context.Background()

	err := PublishEvent(ctx, rc, Event{Type: "x", Data: map[string]any{"bad": make(chan int)}})
	assert.ErrorContains(t, err, "failed to marshal event")

	mr.SetError("ERR redis unavailable")
	err = PublishEvent(ctx, rc, Event{Type: "x"})
	assert.ErrorContains(t, err, "failed to publish event")
}
