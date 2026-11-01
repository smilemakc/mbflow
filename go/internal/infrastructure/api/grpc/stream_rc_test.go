package grpc

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/go/internal/application/engine"
	appobserver "github.com/smilemakc/mbflow/go/internal/application/observer"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// rcStream is an in-process MBFlowServiceAPI_StreamExecutionEventsServer.
type rcStream struct {
	grpc.ServerStream // nil: only Context and Send are used

	ctx     context.Context
	sent    chan *pb.ExecutionEvent
	sendErr error

	mu     sync.Mutex
	events []*pb.ExecutionEvent
}

func rcNewStream(ctx context.Context) *rcStream {
	return &rcStream{ctx: ctx, sent: make(chan *pb.ExecutionEvent, 64)}
}

func (s *rcStream) Context() context.Context { return s.ctx }

func (s *rcStream) Send(ev *pb.ExecutionEvent) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.mu.Lock()
	s.events = append(s.events, ev)
	s.mu.Unlock()
	s.sent <- ev
	return nil
}

func (s *rcStream) all() []*pb.ExecutionEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*pb.ExecutionEvent(nil), s.events...)
}

func rcAwaitSent(t *testing.T, s *rcStream) *pb.ExecutionEvent {
	t.Helper()
	select {
	case ev := <-s.sent:
		return ev
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a streamed event")
		return nil
	}
}

func rcRunStream(srv *ServiceAPIServer, req *pb.StreamExecutionEventsRequest, s *rcStream) <-chan error {
	done := make(chan error, 1)
	go func() { done <- srv.StreamExecutionEvents(req, s) }()
	return done
}

func rcAwaitDone(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("stream handler did not return")
		return nil
	}
}

func rcPersistedExecution(env *rcEnv, status string) uuid.UUID {
	id := uuid.New()
	env.execs.put(&storagemodels.ExecutionModel{ID: id, WorkflowSource: "stored", Status: status})
	return id
}

// ---------------------------------------------------------------------------
// Request validation and lookup failures
// ---------------------------------------------------------------------------

func TestRCStream_RequestValidation(t *testing.T) {
	env := rcNewEnv()
	s := rcNewStream(context.Background())

	err := env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{}, s)
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
	err = env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: "bad"}, s)
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	err = env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: uuid.NewString()}, s)
	assert.Equal(t, codes.NotFound, rcCode(t, err))

	env.execs.setErr("FindByID", errRCBoom)
	err = env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: uuid.NewString()}, s)
	assert.Equal(t, codes.Internal, rcCode(t, err))
}

// ---------------------------------------------------------------------------
// Persisted executions
// ---------------------------------------------------------------------------

func TestRCStream_PersistedOverBufconn(t *testing.T) {
	env := rcNewEnv()
	id := rcPersistedExecution(env, "running")
	env.execs.addEvent(id, 1, "node.started", map[string]any{"node_id": "a"})
	env.execs.addEvent(id, 2, "node.completed", map[string]any{"node_id": "a", "output": map[string]any{"v": 1}})
	env.execs.addEvent(id, 3, "execution.completed", nil)

	lis := bufconn.Listen(1 << 20)
	gs := grpc.NewServer()
	pb.RegisterMBFlowServiceAPIServer(gs, env.srv)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	client := pb.NewMBFlowServiceAPIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := client.StreamExecutionEvents(ctx, &pb.StreamExecutionEventsRequest{ExecutionId: id.String(), AfterSequence: 1})
	require.NoError(t, err)

	var got []*pb.ExecutionEvent
	for {
		ev, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		got = append(got, ev)
	}
	require.Len(t, got, 2, "events up to after_sequence are skipped")
	assert.Equal(t, int64(2), got[0].Sequence)
	assert.Equal(t, "node.completed", got[0].EventType)
	assert.Equal(t, "stored", got[0].WorkflowSource)
	assert.Equal(t, id.String(), got[0].ExecutionId)
	assert.Equal(t, "a", got[0].Payload.AsMap()["node_id"])
	assert.Equal(t, int64(1700000002), got[0].SentAt.AsTime().Unix())
	assert.Equal(t, "execution.completed", got[1].EventType, "stream ends with the terminal event")

	// Invalid request via the real transport
	bad, err := client.StreamExecutionEvents(ctx, &pb.StreamExecutionEventsRequest{})
	require.NoError(t, err)
	_, err = bad.Recv()
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
}

func TestRCStream_PersistedEndsOnTerminalStatus(t *testing.T) {
	env := rcNewEnv()
	id := rcPersistedExecution(env, "failed")
	env.execs.addEvent(id, 1, "node.failed", nil)

	s := rcNewStream(context.Background())
	err := env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: id.String()}, s)
	require.NoError(t, err)
	require.Len(t, s.all(), 1)
	assert.Equal(t, "node.failed", s.all()[0].EventType)
}

func TestRCStream_PersistedPollsForNewEvents(t *testing.T) {
	env := rcNewEnv()
	id := rcPersistedExecution(env, "running")
	s := rcNewStream(context.Background())
	done := rcRunStream(env.srv, &pb.StreamExecutionEventsRequest{ExecutionId: id.String()}, s)

	// Nothing is available on the first poll; the event appears later.
	env.execs.addEvent(id, 1, "execution.cancelled", map[string]any{"reason": "user"})
	ev := rcAwaitSent(t, s)
	assert.Equal(t, "execution.cancelled", ev.EventType)
	require.NoError(t, rcAwaitDone(t, done))
}

func TestRCStream_PersistedStatusChangeEndsStream(t *testing.T) {
	env := rcNewEnv()
	id := rcPersistedExecution(env, "running")
	s := rcNewStream(context.Background())
	done := rcRunStream(env.srv, &pb.StreamExecutionEventsRequest{ExecutionId: id.String()}, s)

	env.execs.setStatus(id, "TIMEOUT") // status comparison is case-insensitive
	require.NoError(t, rcAwaitDone(t, done))
	assert.Empty(t, s.all())
}

func TestRCStream_PersistedClientCancel(t *testing.T) {
	env := rcNewEnv()
	id := rcPersistedExecution(env, "running")
	ctx, cancel := context.WithCancel(context.Background())
	s := rcNewStream(ctx)
	done := rcRunStream(env.srv, &pb.StreamExecutionEventsRequest{ExecutionId: id.String()}, s)
	cancel()
	assert.ErrorIs(t, rcAwaitDone(t, done), context.Canceled)
}

func TestRCStream_PersistedErrors(t *testing.T) {
	t.Run("events load failure", func(t *testing.T) {
		env := rcNewEnv()
		id := rcPersistedExecution(env, "running")
		env.execs.setErr("GetEvents", errRCBoom)
		err := env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: id.String()}, rcNewStream(context.Background()))
		assert.Equal(t, codes.Internal, rcCode(t, err))
	})
	t.Run("status reload failure", func(t *testing.T) {
		env := rcNewEnv()
		id := rcPersistedExecution(env, "running")
		env.execs.failFindByIDAfter = 2
		err := env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: id.String()}, rcNewStream(context.Background()))
		assert.Equal(t, codes.Internal, rcCode(t, err))
	})
	t.Run("send failure", func(t *testing.T) {
		env := rcNewEnv()
		id := rcPersistedExecution(env, "running")
		env.execs.addEvent(id, 1, "node.started", nil)
		s := rcNewStream(context.Background())
		s.sendErr = errors.New("client gone")
		err := env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: id.String()}, s)
		assert.EqualError(t, err, "client gone")
	})
}

func TestRCStream_UsesGetEventsSinceWhenAvailable(t *testing.T) {
	env := rcNewEnv()
	since := &rcEventsSinceRepo{rcExecutionRepo: env.execs}
	env.ops.ExecutionRepo = since
	id := rcPersistedExecution(env, "completed")
	env.execs.addEvent(id, 4, "node.started", nil)
	env.execs.addEvent(id, 5, "execution.completed", nil)

	s := rcNewStream(context.Background())
	require.NoError(t, env.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: id.String(), AfterSequence: 4}, s))
	require.Len(t, s.all(), 1)
	assert.Equal(t, int64(5), s.all()[0].Sequence)
	assert.Equal(t, []int64{4}, since.sinceCalls)
}

func TestRCGetExecutionEventsSince_Filtering(t *testing.T) {
	env := rcNewEnv()
	id := uuid.New()
	env.execs.addEvent(id, 1, "a", nil)
	env.execs.addEvent(id, 2, "b", nil)
	env.execs.addEvent(id, 3, "c", nil)

	all, err := env.srv.getExecutionEventsSince(context.Background(), id, 0)
	require.NoError(t, err)
	assert.Len(t, all, 3)
	tail, err := env.srv.getExecutionEventsSince(context.Background(), id, 2)
	require.NoError(t, err)
	require.Len(t, tail, 1)
	assert.Equal(t, "c", tail[0].EventType)
}

// ---------------------------------------------------------------------------
// Ephemeral (in-memory) executions
// ---------------------------------------------------------------------------

type rcEphemeralEnv struct {
	*rcEnv
	obs      *appobserver.ObserverManager
	registry *engine.EphemeralStreamRegistry
	execID   string
}

func rcNewEphemeralEnv(t *testing.T, withObserver bool) *rcEphemeralEnv {
	t.Helper()
	env := rcNewEnv()
	var obs *appobserver.ObserverManager
	if withObserver {
		obs = appobserver.NewObserverManager()
	}
	registry := engine.NewEphemeralStreamRegistry(time.Minute)
	env.ops.ExecutionMgr = engine.NewExecutionManager(env.executors, env.workflows, env.execs, nil, nil, obs, registry)
	id := uuid.NewString() // not persisted: FindByID fails
	registry.Register(id, engine.NewEphemeralNotifier(obs, engine.NewEventRedactor()))
	return &rcEphemeralEnv{rcEnv: env, obs: obs, registry: registry, execID: id}
}

func (e *rcEphemeralEnv) awaitObserver(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool { return e.obs.Count() == 1 }, 5*time.Second, 5*time.Millisecond)
}

func TestRCStream_EphemeralDeliversObserverEvents(t *testing.T) {
	e := rcNewEphemeralEnv(t, true)
	s := rcNewStream(context.Background())
	done := rcRunStream(e.srv, &pb.StreamExecutionEventsRequest{ExecutionId: e.execID}, s)
	e.awaitObserver(t)

	nodeID, nodeName, nodeType := "n1", "Node", rcNodeType
	wave, count := 0, 1
	dur := int64(12)
	e.obs.Notify(context.Background(), appobserver.Event{
		Type: appobserver.EventTypeNodeCompleted, ExecutionID: e.execID, WorkflowID: "wf", Status: "completed",
		Timestamp: time.Unix(1700000000, 0),
		NodeID:    &nodeID, NodeName: &nodeName, NodeType: &nodeType, WaveIndex: &wave, NodeCount: &count, DurationMs: &dur,
		Input: map[string]any{"in": 1}, Output: map[string]any{"out": 2}, Variables: map[string]any{"v": 3},
		Metadata: map[string]any{"sequence": int64(1)},
	})
	first := rcAwaitSent(t, s)
	assert.Equal(t, int64(1), first.Sequence)
	assert.Equal(t, "inline", first.WorkflowSource)
	assert.Equal(t, string(appobserver.EventTypeNodeCompleted), first.EventType)
	payload := first.Payload.AsMap()
	assert.Equal(t, "n1", payload["node_id"])
	assert.Equal(t, "Node", payload["node_name"])
	assert.EqualValues(t, 12, payload["duration_ms"])
	assert.Equal(t, map[string]any{"out": float64(2)}, payload["output"])

	// Events from other executions are filtered out; a stale sequence is skipped.
	e.obs.Notify(context.Background(), appobserver.Event{Type: appobserver.EventTypeExecutionCompleted, ExecutionID: "other"})
	e.obs.Notify(context.Background(), appobserver.Event{Type: appobserver.EventTypeNodeStarted, ExecutionID: e.execID, Metadata: map[string]any{"sequence": 1}})

	e.obs.Notify(context.Background(), appobserver.Event{
		Type: appobserver.EventTypeExecutionFailed, ExecutionID: e.execID, Error: errors.New("node exploded"),
		Metadata: map[string]any{"sequence": float64(7)},
	})
	last := rcAwaitSent(t, s)
	assert.Equal(t, int64(7), last.Sequence)
	assert.Equal(t, "node exploded", last.Payload.AsMap()["error"])
	require.NoError(t, rcAwaitDone(t, done))
	assert.Len(t, s.all(), 2)
	assert.Zero(t, e.obs.Count(), "observer is unregistered when the stream ends")
}

func TestRCStream_EphemeralAlreadyTerminal(t *testing.T) {
	e := rcNewEphemeralEnv(t, true)
	e.registry.MarkTerminal(e.execID)
	require.NoError(t, e.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: e.execID}, rcNewStream(context.Background())))
}

func TestRCStream_EphemeralBecomesTerminal(t *testing.T) {
	e := rcNewEphemeralEnv(t, true)
	s := rcNewStream(context.Background())
	done := rcRunStream(e.srv, &pb.StreamExecutionEventsRequest{ExecutionId: e.execID}, s)
	e.awaitObserver(t)
	e.registry.MarkTerminal(e.execID)
	require.NoError(t, rcAwaitDone(t, done))
}

func TestRCStream_EphemeralClientCancel(t *testing.T) {
	e := rcNewEphemeralEnv(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	done := rcRunStream(e.srv, &pb.StreamExecutionEventsRequest{ExecutionId: e.execID}, rcNewStream(ctx))
	e.awaitObserver(t)
	cancel()
	assert.ErrorIs(t, rcAwaitDone(t, done), context.Canceled)
}

func TestRCStream_EphemeralSendFailure(t *testing.T) {
	e := rcNewEphemeralEnv(t, true)
	s := rcNewStream(context.Background())
	s.sendErr = errors.New("broken pipe")
	done := rcRunStream(e.srv, &pb.StreamExecutionEventsRequest{ExecutionId: e.execID}, s)
	e.awaitObserver(t)
	e.obs.Notify(context.Background(), appobserver.Event{Type: appobserver.EventTypeNodeStarted, ExecutionID: e.execID})
	assert.EqualError(t, rcAwaitDone(t, done), "broken pipe")
}

func TestRCStream_EphemeralWithoutObserverManager(t *testing.T) {
	e := rcNewEphemeralEnv(t, false)
	err := e.srv.StreamExecutionEvents(&pb.StreamExecutionEventsRequest{ExecutionId: e.execID}, rcNewStream(context.Background()))
	assert.Equal(t, codes.Unavailable, rcCode(t, err))
}

func TestRCStream_EphemeralWithoutExecutionManager(t *testing.T) {
	env := rcNewEnv()
	err := env.srv.streamEphemeralExecutionEvents(context.Background(), &pb.StreamExecutionEventsRequest{ExecutionId: "x"}, rcNewStream(context.Background()))
	assert.Equal(t, codes.Unavailable, rcCode(t, err))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestRCStreamObserver_DropsWhenFull(t *testing.T) {
	o := newStreamExecutionObserver("exec-1")
	assert.Contains(t, o.Name(), "grpc-stream-exec-1-")
	assert.True(t, o.Filter().ShouldNotify(appobserver.Event{ExecutionID: "exec-1"}))
	assert.False(t, o.Filter().ShouldNotify(appobserver.Event{ExecutionID: "exec-2"}))

	for i := 0; i < cap(o.events)+5; i++ {
		require.NoError(t, o.OnEvent(context.Background(), appobserver.Event{ExecutionID: "exec-1"}))
	}
	assert.Len(t, o.events, cap(o.events), "overflowing events are dropped without blocking")
}

func TestRCExtractEventSequence(t *testing.T) {
	ev := func(v any) appobserver.Event {
		return appobserver.Event{Metadata: map[string]any{"sequence": v}}
	}
	assert.Equal(t, int64(9), extractEventSequence(ev(int64(9)), 0))
	assert.Equal(t, int64(8), extractEventSequence(ev(8), 0))
	assert.Equal(t, int64(7), extractEventSequence(ev(7.0), 0))
	assert.Equal(t, int64(6), extractEventSequence(ev(float32(6)), 0))
	assert.Equal(t, int64(4), extractEventSequence(ev("nope"), 3), "unsupported types fall back")
	assert.Equal(t, int64(11), extractEventSequence(appobserver.Event{}, 10))
	assert.Equal(t, int64(1), extractEventSequence(appobserver.Event{Metadata: map[string]any{}}, 0))
}

func TestRCTerminalClassification(t *testing.T) {
	for _, et := range []string{"execution.completed", "execution.failed", "execution.cancelled", "execution.timeout"} {
		assert.True(t, isTerminalEventType(et), et)
	}
	assert.False(t, isTerminalEventType("node.completed"))

	for _, st := range []string{"completed", "FAILED", "cancelled", "timeout"} {
		assert.True(t, isTerminalExecutionStatus(st), st)
	}
	assert.False(t, isTerminalExecutionStatus("running"))
}

func TestRCWorkflowFromProto(t *testing.T) {
	assert.Nil(t, workflowFromProto(nil))
	assert.Nil(t, nodeFromProto(nil))
	assert.Nil(t, edgeFromProto(nil))

	pos, _ := structpb.NewStruct(map[string]any{"x": 10, "y": 20})
	badPos, _ := structpb.NewStruct(map[string]any{"x": 10})
	cond, _ := structpb.NewStruct(map[string]any{"expression": "a == 1"})
	vars, _ := structpb.NewStruct(map[string]any{"k": "v"})
	created := time.Unix(1700000000, 0).UTC()

	wf := workflowFromProto(&pb.Workflow{
		Id: "wf", Name: "W", Description: "D", Version: 3, Status: "active", CreatedBy: "u",
		Variables: vars, CreatedAt: timestamppb.New(created), UpdatedAt: timestamppb.New(created.Add(time.Hour)),
		Nodes: []*pb.Node{
			{Id: "a", Name: "A", Type: rcNodeType, Position: pos},
			{Id: "b", Name: "B", Type: rcNodeType, Position: badPos},
		},
		Edges: []*pb.Edge{{Id: "e", From: "a", To: "b", SourceHandle: "h", Condition: cond, Loop: &pb.EdgeLoopConfig{MaxIterations: 4}}},
	})
	assert.Equal(t, "W", wf.Name)
	assert.Equal(t, 3, wf.Version)
	assert.Equal(t, models.WorkflowStatus("active"), wf.Status)
	assert.Equal(t, "v", wf.Variables["k"])
	assert.True(t, created.Equal(wf.CreatedAt))
	assert.True(t, created.Add(time.Hour).Equal(wf.UpdatedAt))
	require.Len(t, wf.Nodes, 2)
	assert.Equal(t, &models.Position{X: 10, Y: 20}, wf.Nodes[0].Position)
	assert.Nil(t, wf.Nodes[1].Position, "incomplete positions are ignored")
	require.Len(t, wf.Edges, 1)
	assert.Equal(t, "a == 1", wf.Edges[0].Condition)
	assert.Equal(t, "h", wf.Edges[0].SourceHandle)
	assert.Equal(t, 4, wf.Edges[0].Loop.MaxIterations)

	// Round trip through the outbound converter
	back := toProtoEdge(wf.Edges[0])
	assert.Equal(t, "a == 1", back.Condition.AsMap()["expression"])
	assert.Equal(t, int32(4), back.Loop.MaxIterations)
	assert.Nil(t, toProtoEdge(nil))
	assert.Nil(t, toProtoEdge(&models.Edge{ID: "x"}).Condition)
}
