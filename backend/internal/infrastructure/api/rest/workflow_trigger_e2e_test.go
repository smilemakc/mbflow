//go:build integration

package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/migrations"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/pkg/models"
)

// DebugQueryHook logs SQL queries for debugging
type DebugQueryHook struct {
	t *testing.T
}

func (h *DebugQueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

func (h *DebugQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	if event.Err != nil {
		h.t.Logf("SQL ERROR: %v\nQuery: %s", event.Err, event.Query)
	}
}

// MockExternalAPIs encapsulates all mock HTTP servers
type MockExternalAPIs struct {
	ExampleAPI  *MockExampleAPI
	SendGridAPI *MockSendGridAPI
	SegmentAPI  *MockSegmentAPI
}

// MockExampleAPI mocks api.example.com
type MockExampleAPI struct {
	server   *httptest.Server
	mu       sync.Mutex
	profiles []map[string]interface{}
	tasks    []map[string]interface{}
}

func NewMockExampleAPI() *MockExampleAPI {
	mock := &MockExampleAPI{
		profiles: make([]map[string]interface{}, 0),
		tasks:    make([]map[string]interface{}, 0),
	}

	mux := http.NewServeMux()

	// POST /profiles
	mux.HandleFunc("/profiles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var profile map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mock.mu.Lock()
		profile["id"] = len(mock.profiles) + 1
		mock.profiles = append(mock.profiles, profile)
		mock.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"profile": profile,
		})
	})

	// POST /tasks/bulk
	mux.HandleFunc("/tasks/bulk", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mock.mu.Lock()
		mock.tasks = append(mock.tasks, req)
		mock.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       true,
			"tasks_created": 3,
		})
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *MockExampleAPI) URL() string {
	return m.server.URL
}

func (m *MockExampleAPI) Close() {
	m.server.Close()
}

func (m *MockExampleAPI) GetProfiles() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]map[string]interface{}, len(m.profiles))
	copy(result, m.profiles)
	return result
}

func (m *MockExampleAPI) GetTasks() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]map[string]interface{}, len(m.tasks))
	copy(result, m.tasks)
	return result
}

func (m *MockExampleAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles = make([]map[string]interface{}, 0)
	m.tasks = make([]map[string]interface{}, 0)
}

// MockSendGridAPI mocks api.sendgrid.com
type MockSendGridAPI struct {
	server *httptest.Server
	mu     sync.Mutex
	emails []map[string]interface{}
}

func NewMockSendGridAPI() *MockSendGridAPI {
	mock := &MockSendGridAPI{
		emails: make([]map[string]interface{}, 0),
	}

	mux := http.NewServeMux()

	// POST /v3/mail/send
	mux.HandleFunc("/v3/mail/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var email map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&email); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mock.mu.Lock()
		email["message_id"] = fmt.Sprintf("msg_%d", len(mock.emails)+1)
		mock.emails = append(mock.emails, email)
		mock.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"message_id": email["message_id"],
		})
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *MockSendGridAPI) URL() string {
	return m.server.URL
}

func (m *MockSendGridAPI) Close() {
	m.server.Close()
}

func (m *MockSendGridAPI) GetEmails() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]map[string]interface{}, len(m.emails))
	copy(result, m.emails)
	return result
}

func (m *MockSendGridAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emails = make([]map[string]interface{}, 0)
}

// MockSegmentAPI mocks api.segment.com
type MockSegmentAPI struct {
	server *httptest.Server
	mu     sync.Mutex
	events []map[string]interface{}
}

func NewMockSegmentAPI() *MockSegmentAPI {
	mock := &MockSegmentAPI{
		events: make([]map[string]interface{}, 0),
	}

	mux := http.NewServeMux()

	// POST /v1/track
	mux.HandleFunc("/v1/track", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var event map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mock.mu.Lock()
		event["tracked_at"] = time.Now().Unix()
		mock.events = append(mock.events, event)
		mock.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *MockSegmentAPI) URL() string {
	return m.server.URL
}

func (m *MockSegmentAPI) Close() {
	m.server.Close()
}

func (m *MockSegmentAPI) GetEvents() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]map[string]interface{}, len(m.events))
	copy(result, m.events)
	return result
}

func (m *MockSegmentAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]map[string]interface{}, 0)
}

// E2ETestEnvironment encapsulates the entire test environment
type E2ETestEnvironment struct {
	DB            *bun.DB
	Pool          *dockertest.Pool
	PostgresRes   *dockertest.Resource
	Mocks         *MockExternalAPIs
	WorkflowRepo  *storage.WorkflowRepository
	ExecutionRepo *storage.ExecutionRepository
	ExecutorMgr   executor.Manager
	ExecutionMgr  *engine.ExecutionManager
	WorkflowDef   map[string]interface{}
}

// setupE2EEnvironment sets up the complete E2E test environment
func setupE2EEnvironment(t *testing.T) *E2ETestEnvironment {
	t.Helper()

	env := &E2ETestEnvironment{}

	// Setup Dockertest with explicit Docker endpoint
	var pool *dockertest.Pool
	var err error

	// Determine Docker endpoint
	dockerEndpoint := os.Getenv("DOCKER_HOST")
	if dockerEndpoint == "" {
		// Try macOS Docker Desktop socket
		macOSSocket := os.Getenv("HOME") + "/.docker/run/docker.sock"
		if _, statErr := os.Stat(macOSSocket); statErr == nil {
			dockerEndpoint = "unix://" + macOSSocket
		}
	}

	pool, err = dockertest.NewPool(dockerEndpoint)
	require.NoError(t, err, "Failed to connect to Docker. Is Docker running? Tried endpoint: %s", dockerEndpoint)

	// Verify Docker is accessible
	err = pool.Client.Ping()
	require.NoError(t, err, "Failed to ping Docker daemon")
	env.Pool = pool
	// pgPort := "5676"
	// Start PostgreSQL 16
	env.PostgresRes, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_USER=mbflow_test",
			"POSTGRES_PASSWORD=mbflow_test",
			"POSTGRES_DB=mbflow_test",
		},
		// PortBindings: map[docker.Port][]docker.PortBinding{
		// 	"5432": {{HostPort: pgPort}},
		// },
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(t, err)

	// Set expiry for the container
	env.PostgresRes.Expire(300) // 5 minutes

	// Wait for PostgreSQL to be ready
	var db *bun.DB
	err = pool.Retry(func() error {
		dsn := fmt.Sprintf("postgres://mbflow_test:mbflow_test@localhost:%s/mbflow_test?sslmode=disable",
			env.PostgresRes.GetPort("5432/tcp"))

		connector := pgdriver.NewConnector(
			pgdriver.WithDSN(dsn),
			pgdriver.WithTimeout(5*time.Second),
		)
		sqldb := sql.OpenDB(connector)
		db = bun.NewDB(sqldb, pgdialect.New())

		// Enable query logging for debugging
		db.WithQueryHook(&DebugQueryHook{t: t})

		return db.Ping()
	})
	require.NoError(t, err)
	env.DB = db

	// Run migrations using embedded migration files
	migrator, err := storage.NewMigrator(env.DB, migrations.FS)
	require.NoError(t, err)

	err = migrator.Init(context.Background())
	require.NoError(t, err)

	err = migrator.Up(context.Background())
	require.NoError(t, err)

	// Setup repositories
	env.WorkflowRepo = storage.NewWorkflowRepository(env.DB)
	env.ExecutionRepo = storage.NewExecutionRepository(env.DB)

	// Setup executor manager
	env.ExecutorMgr = executor.NewManager()
	err = builtin.RegisterBuiltins(env.ExecutorMgr)
	require.NoError(t, err)

	// Setup execution manager
	env.ExecutionMgr = engine.NewExecutionManager(
		env.ExecutorMgr,
		env.WorkflowRepo,
		env.ExecutionRepo,
		nil, // eventRepo - deferred for MVP
		nil, // resourceRepo - optional for tests
		nil, // observerManager - optional for tests
	)

	// Setup mock APIs
	env.Mocks = &MockExternalAPIs{
		ExampleAPI:  NewMockExampleAPI(),
		SendGridAPI: NewMockSendGridAPI(),
		SegmentAPI:  NewMockSegmentAPI(),
	}

	// Load workflow definition from fixture
	fixturePath := "../../../../test/fixtures/user_onboarding_test.json"
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		fixturePath = "../../../test/fixtures/user_onboarding_test.json"
	}
	fixtureData, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	var workflowDef map[string]interface{}
	err = json.Unmarshal(fixtureData, &workflowDef)
	require.NoError(t, err)
	env.WorkflowDef = workflowDef

	return env
}

// cleanup tears down the E2E test environment
func (env *E2ETestEnvironment) cleanup(t *testing.T) {
	t.Helper()

	if env.Mocks != nil {
		if env.Mocks.ExampleAPI != nil {
			env.Mocks.ExampleAPI.Close()
		}
		if env.Mocks.SendGridAPI != nil {
			env.Mocks.SendGridAPI.Close()
		}
		if env.Mocks.SegmentAPI != nil {
			env.Mocks.SegmentAPI.Close()
		}
	}

	if env.DB != nil {
		env.DB.Close()
	}

	if env.PostgresRes != nil && env.Pool != nil {
		env.Pool.Purge(env.PostgresRes)
	}
}

// createWorkflowFromFixture creates a workflow from the test fixture
func (env *E2ETestEnvironment) createWorkflowFromFixture(t *testing.T, ctx context.Context) *storagemodels.WorkflowModel {
	t.Helper()

	workflowData := env.WorkflowDef["workflow"].(map[string]interface{})

	// Replace template placeholders with mock URLs
	workflowJSON, err := json.Marshal(workflowData)
	require.NoError(t, err)

	workflowJSONStr := string(workflowJSON)
	workflowJSONStr = strings.ReplaceAll(workflowJSONStr, "{{MOCK_EXAMPLE_API}}", env.Mocks.ExampleAPI.URL())
	workflowJSONStr = strings.ReplaceAll(workflowJSONStr, "{{MOCK_SENDGRID_API}}", env.Mocks.SendGridAPI.URL())
	workflowJSONStr = strings.ReplaceAll(workflowJSONStr, "{{MOCK_SEGMENT_API}}", env.Mocks.SegmentAPI.URL())

	err = json.Unmarshal([]byte(workflowJSONStr), &workflowData)
	require.NoError(t, err)

	// Create workflow model
	variables := storagemodels.JSONBMap{}
	if vars, ok := workflowData["variables"].(map[string]interface{}); ok && vars != nil {
		variables = storagemodels.JSONBMap(vars)
	}

	workflowModel := &storagemodels.WorkflowModel{
		ID:          uuid.New(),
		Name:        workflowData["name"].(string),
		Description: workflowData["description"].(string),
		Status:      workflowData["status"].(string),
		Version:     1,
		Variables:   variables,
		Metadata:    storagemodels.JSONBMap{}, // Initialize empty metadata
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Add nodes
	nodesData := workflowData["nodes"].([]interface{})
	for _, nodeData := range nodesData {
		nodeMap := nodeData.(map[string]interface{})

		// Get config, ensure it's not nil
		config := storagemodels.JSONBMap{}
		if cfgData, ok := nodeMap["config"].(map[string]interface{}); ok && cfgData != nil {
			config = storagemodels.JSONBMap(cfgData)
		}

		nodeModel := &storagemodels.NodeModel{
			ID:         uuid.New(),
			WorkflowID: workflowModel.ID,
			NodeID:     nodeMap["id"].(string),
			Name:       nodeMap["name"].(string),
			Type:       nodeMap["type"].(string),
			Config:     config,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		workflowModel.Nodes = append(workflowModel.Nodes, nodeModel)
	}

	// Add edges
	edgesData := workflowData["edges"].([]interface{})
	for _, edgeData := range edgesData {
		edgeMap := edgeData.(map[string]interface{})
		edgeModel := &storagemodels.EdgeModel{
			ID:         uuid.New(),
			WorkflowID: workflowModel.ID,
			EdgeID:     edgeMap["id"].(string),
			FromNodeID: edgeMap["from"].(string),
			ToNodeID:   edgeMap["to"].(string),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		workflowModel.Edges = append(workflowModel.Edges, edgeModel)
	}

	// Save to database
	err = env.WorkflowRepo.Create(ctx, workflowModel)
	require.NoError(t, err)

	return workflowModel
}

// publishUserCreatedEvent simulates publishing a user.created event
func publishUserCreatedEvent(ctx context.Context, executionMgr *engine.ExecutionManager, workflowID uuid.UUID, eventData map[string]interface{}) (*models.Execution, error) {
	// In a real implementation, this would go through the event trigger system
	// For now, we directly execute the workflow with the event data
	opts := engine.DefaultExecutionOptions()
	return executionMgr.Execute(ctx, workflowID.String(), eventData, opts)
}

// waitForExecution waits for an execution to complete or timeout
func waitForExecution(t *testing.T, ctx context.Context, repo *storage.ExecutionRepository, executionID uuid.UUID, timeout time.Duration) *storagemodels.ExecutionModel {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("context cancelled while waiting for execution")
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Fatal("timeout waiting for execution to complete")
			}

			exec, err := repo.FindByIDWithRelations(ctx, executionID)
			if err != nil {
				continue
			}

			if exec.Status == "completed" || exec.Status == "failed" {
				return exec
			}
		}
	}
}

// waitForMockData waits for mock data to be available with retry logic
// This handles race conditions where HTTP responses might still be in-flight
// even after execution is marked as completed in the database
func waitForMockData(t *testing.T, getter func() int, expectedCount int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		count := getter()
		if count >= expectedCount {
			return
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Fatalf("timeout waiting for mock data: expected %d, got %d", expectedCount, count)
			}
		}
	}
}

// Test 1: Happy Path - Complete workflow execution
func TestUserOnboardingWorkflow_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	env := setupE2EEnvironment(t)
	defer env.cleanup(t)

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(t, ctx)

	// Reset mocks
	env.Mocks.ExampleAPI.Reset()
	env.Mocks.SendGridAPI.Reset()
	env.Mocks.SegmentAPI.Reset()

	// Publish user.created event
	eventData := map[string]interface{}{
		"user_id":    "usr_12345",
		"email":      "john.doe@example.com",
		"name":       "John Doe",
		"status":     "active",
		"source":     "api",
		"created_at": time.Now().Format(time.RFC3339),
	}

	execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
	require.NoError(t, err)
	require.NotNil(t, execution)

	// Wait for execution to complete
	execUUID, err := uuid.Parse(execution.ID)
	require.NoError(t, err)

	execModel := waitForExecution(t, ctx, env.ExecutionRepo, execUUID, 10*time.Second)
	assert.Equal(t, "completed", execModel.Status)

	// Wait for all mock API calls to be processed (handles race conditions)
	waitForMockData(t, func() int { return len(env.Mocks.ExampleAPI.GetProfiles()) }, 1, 2*time.Second)
	waitForMockData(t, func() int { return len(env.Mocks.SendGridAPI.GetEmails()) }, 1, 2*time.Second)
	waitForMockData(t, func() int { return len(env.Mocks.SegmentAPI.GetEvents()) }, 1, 2*time.Second)
	waitForMockData(t, func() int { return len(env.Mocks.ExampleAPI.GetTasks()) }, 1, 2*time.Second)

	// Verify all API calls were made
	profiles := env.Mocks.ExampleAPI.GetProfiles()
	assert.Len(t, profiles, 1, "should create one profile")
	assert.Equal(t, "usr_12345", profiles[0]["user_id"])

	emails := env.Mocks.SendGridAPI.GetEmails()
	assert.Len(t, emails, 1, "should send one email")

	events := env.Mocks.SegmentAPI.GetEvents()
	assert.Len(t, events, 1, "should track one event")
	assert.Equal(t, "Onboarding Started", events[0]["event"])

	tasks := env.Mocks.ExampleAPI.GetTasks()
	assert.Len(t, tasks, 1, "should create onboarding tasks")
}

// Test 2: Event Filtering - Should NOT Trigger
// TODO: Event filtering is not yet implemented in the trigger system.
// This test currently bypasses the trigger and directly calls the execution manager,
// so it will execute the workflow regardless of filter conditions.
// Once event filtering is implemented, update this test to verify that
// executions are NOT created when filter conditions don't match.
func TestUserOnboardingWorkflow_EventFiltering_ShouldNotTrigger(t *testing.T) {
	t.Skip("TODO: Event filtering not implemented yet - see trigger system roadmap")

	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tests := []struct {
		name      string
		eventData map[string]interface{}
		reason    string
	}{
		{
			name: "wrong source",
			eventData: map[string]interface{}{
				"user_id":    "usr_99999",
				"email":      "test@example.com",
				"name":       "Test User",
				"status":     "active",
				"source":     "webhook", // Should be "api"
				"created_at": time.Now().Format(time.RFC3339),
			},
			reason: "source is webhook, not api",
		},
		{
			name: "wrong status",
			eventData: map[string]interface{}{
				"user_id":    "usr_88888",
				"email":      "pending@example.com",
				"name":       "Pending User",
				"status":     "pending", // Should be "active"
				"source":     "api",
				"created_at": time.Now().Format(time.RFC3339),
			},
			reason: "status is pending, not active",
		},
		{
			name: "both wrong",
			eventData: map[string]interface{}{
				"user_id":    "usr_77777",
				"email":      "invalid@example.com",
				"name":       "Invalid User",
				"status":     "pending",
				"source":     "webhook",
				"created_at": time.Now().Format(time.RFC3339),
			},
			reason: "both source and status are wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupE2EEnvironment(t)
			defer env.cleanup(t)

			ctx := context.Background()

			// Create workflow
			workflowModel := env.createWorkflowFromFixture(t, ctx)

			// Reset mocks
			env.Mocks.ExampleAPI.Reset()
			env.Mocks.SendGridAPI.Reset()
			env.Mocks.SegmentAPI.Reset()

			// Note: In a real event trigger system, this event would be filtered out
			// and the workflow would not execute. For this test, we're validating
			// that if the execution happens, it processes the data correctly.
			// A full event trigger system would be tested separately.

			execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, tt.eventData)
			require.NoError(t, err)

			// The workflow executes with the provided data
			// In production, the trigger filter would prevent execution
			execUUID, err := uuid.Parse(execution.ID)
			require.NoError(t, err)

			execModel := waitForExecution(t, ctx, env.ExecutionRepo, execUUID, 10*time.Second)
			assert.Equal(t, "completed", execModel.Status, "workflow should complete even with filtered data")

			// Verify data was passed through correctly
			profiles := env.Mocks.ExampleAPI.GetProfiles()
			assert.Len(t, profiles, 1)
		})
	}
}

// Test 3: Event Filtering - Should Trigger
func TestUserOnboardingWorkflow_EventFiltering_ShouldTrigger(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	env := setupE2EEnvironment(t)
	defer env.cleanup(t)

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(t, ctx)

	// Reset mocks
	env.Mocks.ExampleAPI.Reset()
	env.Mocks.SendGridAPI.Reset()
	env.Mocks.SegmentAPI.Reset()

	// Event that matches filter: source=api AND status=active
	eventData := map[string]interface{}{
		"user_id":    "usr_11111",
		"email":      "valid@example.com",
		"name":       "Valid User",
		"status":     "active",
		"source":     "api",
		"created_at": time.Now().Format(time.RFC3339),
	}

	execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
	require.NoError(t, err)

	execUUID, err := uuid.Parse(execution.ID)
	require.NoError(t, err)

	execModel := waitForExecution(t, ctx, env.ExecutionRepo, execUUID, 10*time.Second)
	assert.Equal(t, "completed", execModel.Status)

	// Verify all APIs were called
	assert.Len(t, env.Mocks.ExampleAPI.GetProfiles(), 1)
	assert.Len(t, env.Mocks.SendGridAPI.GetEmails(), 1)
	assert.Len(t, env.Mocks.SegmentAPI.GetEvents(), 1)
}

// Test 4: Template Resolution
func TestUserOnboardingWorkflow_TemplateResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	env := setupE2EEnvironment(t)
	defer env.cleanup(t)

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(t, ctx)

	// Reset mocks
	env.Mocks.ExampleAPI.Reset()
	env.Mocks.SendGridAPI.Reset()
	env.Mocks.SegmentAPI.Reset()

	// Event with specific values to verify template resolution
	eventData := map[string]interface{}{
		"user_id":    "usr_template_test",
		"email":      "template@example.com",
		"name":       "Template User",
		"status":     "active",
		"source":     "api",
		"created_at": "2024-01-01T12:00:00Z",
	}

	execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
	require.NoError(t, err)

	execUUID, err := uuid.Parse(execution.ID)
	require.NoError(t, err)

	execModel := waitForExecution(t, ctx, env.ExecutionRepo, execUUID, 10*time.Second)
	assert.Equal(t, "completed", execModel.Status)

	// Verify template variables were resolved correctly
	profiles := env.Mocks.ExampleAPI.GetProfiles()
	require.Len(t, profiles, 1)
	assert.Equal(t, "usr_template_test", profiles[0]["user_id"])
	assert.Equal(t, "template@example.com", profiles[0]["email"])
	assert.Equal(t, "Template User", profiles[0]["name"])
	assert.Equal(t, "2024-01-01T12:00:00Z", profiles[0]["created_at"])

	// Verify email personalization
	emails := env.Mocks.SendGridAPI.GetEmails()
	require.Len(t, emails, 1)
	personalizations := emails[0]["personalizations"].([]interface{})
	require.Len(t, personalizations, 1)
	personalization := personalizations[0].(map[string]interface{})
	to := personalization["to"].([]interface{})
	require.Len(t, to, 1)
	toEmail := to[0].(map[string]interface{})
	assert.Equal(t, "template@example.com", toEmail["email"])

	// Verify event tracking
	events := env.Mocks.SegmentAPI.GetEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "usr_template_test", events[0]["userId"])
	properties := events[0]["properties"].(map[string]interface{})
	assert.Equal(t, "api", properties["source"])
}

// Test 5: Parallel Execution
func TestUserOnboardingWorkflow_ParallelExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	env := setupE2EEnvironment(t)
	defer env.cleanup(t)

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(t, ctx)

	// Reset mocks
	env.Mocks.ExampleAPI.Reset()
	env.Mocks.SendGridAPI.Reset()
	env.Mocks.SegmentAPI.Reset()

	eventData := map[string]interface{}{
		"user_id":    "usr_parallel",
		"email":      "parallel@example.com",
		"name":       "Parallel User",
		"status":     "active",
		"source":     "api",
		"created_at": time.Now().Format(time.RFC3339),
	}

	execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
	require.NoError(t, err)

	execUUID, err := uuid.Parse(execution.ID)
	require.NoError(t, err)

	execModel := waitForExecution(t, ctx, env.ExecutionRepo, execUUID, 10*time.Second)
	assert.Equal(t, "completed", execModel.Status)

	// Verify parallel execution: send_welcome_email and create_onboarding_tasks
	// should execute in parallel after create_profile
	assert.Len(t, execModel.NodeExecutions, 4, "should have 4 node executions")

	// Verify execution order: create_profile should be first
	// Build node ID to logical ID map
	nodeIDMap := make(map[uuid.UUID]string)
	for _, node := range workflowModel.Nodes {
		nodeIDMap[node.ID] = node.NodeID
	}

	var createProfileExec *storagemodels.NodeExecutionModel
	for _, ne := range execModel.NodeExecutions {
		// Find the node execution for create_profile
		if logicalID, ok := nodeIDMap[ne.NodeID]; ok && logicalID == "create_profile" {
			createProfileExec = ne
			break
		}
	}
	require.NotNil(t, createProfileExec, "create_profile execution should exist")

	// Verify all nodes completed successfully
	for _, ne := range execModel.NodeExecutions {
		assert.Equal(t, "completed", ne.Status, "all nodes should complete successfully")
	}
}

// Test 6: Execution Order
func TestUserOnboardingWorkflow_ExecutionOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	env := setupE2EEnvironment(t)
	defer env.cleanup(t)

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(t, ctx)

	// Reset mocks
	env.Mocks.ExampleAPI.Reset()
	env.Mocks.SendGridAPI.Reset()
	env.Mocks.SegmentAPI.Reset()

	eventData := map[string]interface{}{
		"user_id":    "usr_order",
		"email":      "order@example.com",
		"name":       "Order User",
		"status":     "active",
		"source":     "api",
		"created_at": time.Now().Format(time.RFC3339),
	}

	execution, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
	require.NoError(t, err)

	execUUID, err := uuid.Parse(execution.ID)
	require.NoError(t, err)

	execModel := waitForExecution(t, ctx, env.ExecutionRepo, execUUID, 10*time.Second)
	assert.Equal(t, "completed", execModel.Status)

	// Expected order:
	// 1. create_profile (first wave)
	// 2. send_welcome_email and create_onboarding_tasks (second wave, parallel)
	// 3. track_event (third wave, after send_welcome_email)

	// Build node ID to logical ID map
	nodeIDMap := make(map[uuid.UUID]string)
	for _, node := range workflowModel.Nodes {
		nodeIDMap[node.ID] = node.NodeID
	}

	// Get execution times
	executionTimes := make(map[string]time.Time)
	for _, ne := range execModel.NodeExecutions {
		if logicalID, ok := nodeIDMap[ne.NodeID]; ok && ne.StartedAt != nil {
			executionTimes[logicalID] = *ne.StartedAt
		}
	}

	// Verify create_profile executed first
	assert.True(t, executionTimes["create_profile"].Before(executionTimes["send_welcome_email"]))
	assert.True(t, executionTimes["create_profile"].Before(executionTimes["create_onboarding_tasks"]))

	// Verify track_event executed after send_welcome_email
	assert.True(t, executionTimes["send_welcome_email"].Before(executionTimes["track_event"]))
}

// Test 7: Error Handling
func TestUserOnboardingWorkflow_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	env := setupE2EEnvironment(t)
	defer env.cleanup(t)

	ctx := context.Background()

	// Create workflow with invalid URL to trigger error
	workflowModel := env.createWorkflowFromFixture(t, ctx)

	// Modify the first node to point to invalid URL
	for _, node := range workflowModel.Nodes {
		if node.NodeID == "create_profile" {
			config := node.Config
			config["url"] = "http://invalid-host-that-does-not-exist.local/profiles"
			node.Config = config
			_, err := env.DB.NewUpdate().
				Model(node).
				Column("config").
				Where("id = ?", node.ID).
				Exec(ctx)
			require.NoError(t, err)
			break
		}
	}

	eventData := map[string]interface{}{
		"user_id":    "usr_error",
		"email":      "error@example.com",
		"name":       "Error User",
		"status":     "active",
		"source":     "api",
		"created_at": time.Now().Format(time.RFC3339),
	}

	execution, execErr := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
	// The execution manager returns an error if the workflow fails, but we still get the execution object
	require.NotNil(t, execution, "execution should be created even if it fails")
	assert.Error(t, execErr, "publishUserCreatedEvent should return error when workflow fails")

	execUUID, err := uuid.Parse(execution.ID)
	require.NoError(t, err)

	// Wait longer for error case (HTTP timeout + retry logic)
	execModel := waitForExecution(t, ctx, env.ExecutionRepo, execUUID, 30*time.Second)
	assert.Equal(t, "failed", execModel.Status, "execution should fail due to invalid URL")

	// Verify error was captured in execution
	assert.NotEmpty(t, execModel.Error, "execution should have error message")

	// Verify at least one node execution has an error
	hasError := false
	for _, ne := range execModel.NodeExecutions {
		if ne.Status == "failed" && ne.Error != "" {
			hasError = true
			// Error should be network-related (DNS lookup failure or timeout)
			assert.Contains(t, ne.Error, "invalid-host-that-does-not-exist.local",
				"error should mention the invalid hostname")
			break
		}
	}
	assert.True(t, hasError, "at least one node execution should have failed with error")
}

// Test 8: Concurrent Events
func TestUserOnboardingWorkflow_ConcurrentEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	env := setupE2EEnvironment(t)
	defer env.cleanup(t)

	ctx := context.Background()

	// Create workflow
	workflowModel := env.createWorkflowFromFixture(t, ctx)

	// Reset mocks
	env.Mocks.ExampleAPI.Reset()
	env.Mocks.SendGridAPI.Reset()
	env.Mocks.SegmentAPI.Reset()

	// Publish 10 events concurrently
	numEvents := 10
	var wg sync.WaitGroup
	executions := make([]*models.Execution, numEvents)
	errors := make([]error, numEvents)

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			eventData := map[string]interface{}{
				"user_id":    fmt.Sprintf("usr_concurrent_%d", idx),
				"email":      fmt.Sprintf("user%d@example.com", idx),
				"name":       fmt.Sprintf("User %d", idx),
				"status":     "active",
				"source":     "api",
				"created_at": time.Now().Format(time.RFC3339),
			}

			exec, err := publishUserCreatedEvent(ctx, env.ExecutionMgr, workflowModel.ID, eventData)
			executions[idx] = exec
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify all executions started successfully
	for i := 0; i < numEvents; i++ {
		require.NoError(t, errors[i], "execution %d should start without error", i)
		require.NotNil(t, executions[i], "execution %d should not be nil", i)
	}

	// Wait for all executions to complete
	for i := 0; i < numEvents; i++ {
		execUUID, err := uuid.Parse(executions[i].ID)
		require.NoError(t, err)

		execModel := waitForExecution(t, ctx, env.ExecutionRepo, execUUID, 15*time.Second)
		assert.Equal(t, "completed", execModel.Status, "execution %d should complete", i)
	}

	// Wait for all mock API calls to be processed
	waitForMockData(t, func() int { return len(env.Mocks.ExampleAPI.GetProfiles()) }, numEvents, 5*time.Second)
	waitForMockData(t, func() int { return len(env.Mocks.SendGridAPI.GetEmails()) }, numEvents, 5*time.Second)
	waitForMockData(t, func() int { return len(env.Mocks.SegmentAPI.GetEvents()) }, numEvents, 5*time.Second)

	// Verify all API calls were made
	profiles := env.Mocks.ExampleAPI.GetProfiles()
	assert.Len(t, profiles, numEvents, "should create %d profiles", numEvents)

	emails := env.Mocks.SendGridAPI.GetEmails()
	assert.Len(t, emails, numEvents, "should send %d emails", numEvents)

	events := env.Mocks.SegmentAPI.GetEvents()
	assert.Len(t, events, numEvents, "should track %d events", numEvents)

	// Verify data isolation - each user's data should be distinct
	userIDs := make(map[string]bool)
	for _, profile := range profiles {
		userID, ok := profile["user_id"].(string)
		assert.True(t, ok, "user_id should be a string")
		assert.NotEmpty(t, userID, "user_id should not be empty")
		assert.False(t, userIDs[userID], "user_id %s should be unique", userID)
		userIDs[userID] = true

		// Verify it's one of our expected user IDs
		assert.Regexp(t, `^usr_concurrent_\d+$`, userID, "user_id should match pattern")
	}

	// Verify all 10 unique users were created
	assert.Len(t, userIDs, numEvents, "should have %d unique users", numEvents)

	// Verify emails were sent to different addresses
	emailAddresses := make(map[string]bool)
	for _, email := range emails {
		// SendGrid email structure has personalizations array
		personalizations, ok := email["personalizations"].([]interface{})
		if assert.True(t, ok, "email should have personalizations") && len(personalizations) > 0 {
			p := personalizations[0].(map[string]interface{})
			to, ok := p["to"].([]interface{})
			if assert.True(t, ok, "personalization should have to array") && len(to) > 0 {
				toAddr := to[0].(map[string]interface{})
				emailAddr, ok := toAddr["email"].(string)
				assert.True(t, ok, "to should have email string")
				assert.NotEmpty(t, emailAddr, "email address should not be empty")
				emailAddresses[emailAddr] = true
			}
		}
	}
	assert.Len(t, emailAddresses, numEvents, "should have %d unique email addresses", numEvents)
}
