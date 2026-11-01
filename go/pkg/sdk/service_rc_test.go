package sdk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

type rcServiceCall struct {
	Method     string
	Path       string
	RawQuery   string
	SystemKey  string
	OnBehalfOf string
	Body       map[string]any
}

// rcServiceServer records Service API calls and answers with a fixed status/body.
type rcServiceServer struct {
	mu     sync.Mutex
	calls  []rcServiceCall
	status int
	body   string
}

func rcNewServiceServer(t *testing.T) (*rcServiceServer, *ServiceClient) {
	t.Helper()
	s := &rcServiceServer{status: http.StatusOK, body: `{}`}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		call := rcServiceCall{
			Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery,
			SystemKey: r.Header.Get("X-System-Key"), OnBehalfOf: r.Header.Get("X-On-Behalf-Of"),
		}
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			_ = json.Unmarshal(b, &call.Body)
		}
		s.calls = append(s.calls, call)
		w.WriteHeader(s.status)
		_, _ = w.Write([]byte(s.body))
	}))
	t.Cleanup(srv.Close)

	c, err := NewServiceClient(ServiceClientConfig{Endpoint: srv.URL, SystemKey: "sysk_test"})
	require.NoError(t, err)
	return s, c
}

func (s *rcServiceServer) respond(status int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status, s.body = status, body
}

func (s *rcServiceServer) last() rcServiceCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[len(s.calls)-1]
}

func TestRCServiceWorkflows(t *testing.T) {
	s, c := rcNewServiceServer(t)
	ctx := context.Background()

	// The server's list envelope uses "data".
	s.respond(http.StatusOK, `{"data":[{"id":"w1"},{"id":"w2"}],"total":7,"limit":2,"offset":0}`)
	list, total, err := c.Workflows.List(ctx, &ListOptions{Limit: 2, Status: "active"})
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, 7, total)
	call := s.last()
	assert.Equal(t, "GET /api/v1/service/workflows", call.Method+" "+call.Path)
	assert.Equal(t, "limit=2&offset=0&status=active", call.RawQuery)
	assert.Equal(t, "sysk_test", call.SystemKey)
	assert.Empty(t, call.OnBehalfOf)

	// Legacy resource-named key is still accepted.
	s.respond(http.StatusOK, `{"workflows":[{"id":"w3"}],"total":1}`)
	list, _, err = c.Workflows.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "w3", list[0].ID)
	assert.Empty(t, s.last().RawQuery)

	s.respond(http.StatusOK, `{"id":"w1","name":"n"}`)
	wf, err := c.Workflows.Get(ctx, "w1", OnBehalfOf("user-9"))
	require.NoError(t, err)
	assert.Equal(t, "n", wf.Name)
	assert.Equal(t, "user-9", s.last().OnBehalfOf)

	s.respond(http.StatusCreated, `{"id":"w4","name":"created"}`)
	wf, err = c.Workflows.Create(ctx, &ServiceCreateWorkflowRequest{Name: "created"})
	require.NoError(t, err)
	assert.Equal(t, "w4", wf.ID)
	assert.Equal(t, "created", s.last().Body["name"])

	s.respond(http.StatusOK, `{"id":"w4","name":"updated"}`)
	wf, err = c.Workflows.Update(ctx, "w4", &ServiceUpdateWorkflowRequest{
		Name:  "updated",
		Nodes: []ServiceNodeRequest{{ID: "n", Name: "N", Type: "http"}},
		Edges: []ServiceEdgeRequest{{ID: "e", From: "n", To: "m", Condition: map[string]any{"expression": "x"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, "updated", wf.Name)
	assert.Equal(t, "PUT", s.last().Method)

	s.respond(http.StatusOK, `{"message":"deleted"}`)
	require.NoError(t, c.Workflows.Delete(ctx, "w4"))
	assert.Equal(t, "DELETE /api/v1/service/workflows/w4", s.last().Method+" "+s.last().Path)

	s.respond(http.StatusNotFound, `{"code":"WORKFLOW_NOT_FOUND","message":"workflow not found"}`)
	_, err = c.Workflows.Get(ctx, "missing")
	assert.EqualError(t, err, "API error WORKFLOW_NOT_FOUND: workflow not found (status 404)")
	err = c.Workflows.Delete(ctx, "missing")
	assert.EqualError(t, err, "API error WORKFLOW_NOT_FOUND: workflow not found (status 404)")
	_, _, err = c.Workflows.List(ctx, nil)
	assert.EqualError(t, err, "API error WORKFLOW_NOT_FOUND: workflow not found (status 404)")

	s.respond(http.StatusInternalServerError, `oops`)
	_, err = c.Workflows.Create(ctx, &ServiceCreateWorkflowRequest{Name: "x"})
	assert.EqualError(t, err, "API error (status 500)")
	err = c.Workflows.Delete(ctx, "x")
	assert.EqualError(t, err, "API error (status 500)")
	_, _, err = c.Workflows.List(ctx, nil)
	assert.EqualError(t, err, "API error (status 500)")
	_, err = c.Workflows.Update(ctx, "x", &ServiceUpdateWorkflowRequest{})
	assert.Error(t, err)

	s.respond(http.StatusOK, `not-json`)
	_, err = c.Workflows.Get(ctx, "x")
	assert.ErrorContains(t, err, "failed to decode response")
	_, _, err = c.Workflows.List(ctx, nil)
	assert.ErrorContains(t, err, "failed to decode response")

	s.respond(http.StatusOK, `{"data":{"bad":true}}`)
	_, _, err = c.Workflows.List(ctx, nil)
	assert.ErrorContains(t, err, "failed to decode items")
}

func TestRCServiceExecutions(t *testing.T) {
	s, c := rcNewServiceServer(t)
	ctx := context.Background()
	c = c.As("impersonated")

	s.respond(http.StatusOK, `{"data":[{"id":"e1","status":"completed"}],"total":1}`)
	list, total, err := c.Executions.List(ctx, &ServiceExecutionListOptions{Limit: 5, Offset: 10, WorkflowID: "w1", Status: "completed"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, models.ExecutionStatusCompleted, list[0].Status)
	assert.Equal(t, "limit=5&offset=10&workflow_id=w1&status=completed", s.last().RawQuery)
	assert.Equal(t, "impersonated", s.last().OnBehalfOf)

	s.respond(http.StatusOK, `{"id":"e1"}`)
	ex, err := c.Executions.Get(ctx, "e1")
	require.NoError(t, err)
	assert.Equal(t, "e1", ex.ID)

	s.respond(http.StatusAccepted, `{"id":"e2","status":"pending"}`)
	ex, err = c.Executions.Start(ctx, "w1", map[string]any{"k": "v"})
	require.NoError(t, err)
	assert.Equal(t, "e2", ex.ID)
	assert.Equal(t, "POST /api/v1/service/workflows/w1/execute", s.last().Method+" "+s.last().Path)
	assert.Equal(t, map[string]any{"input": map[string]any{"k": "v"}}, s.last().Body)

	s.respond(http.StatusOK, `{}`)
	require.NoError(t, c.Executions.Cancel(ctx, "e1"))
	assert.Equal(t, "/api/v1/service/executions/e1/cancel", s.last().Path)
	require.NoError(t, c.Executions.Retry(ctx, "e1"))
	assert.Equal(t, "/api/v1/service/executions/e1/retry", s.last().Path)

	s.respond(http.StatusConflict, `{"code":"C","message":"m"}`)
	assert.Error(t, c.Executions.Cancel(ctx, "e1"))
	assert.Error(t, c.Executions.Retry(ctx, "e1"))
	_, err = c.Executions.Start(ctx, "w1", nil)
	assert.Error(t, err)
	_, err = c.Executions.Get(ctx, "e1")
	assert.Error(t, err)
	_, _, err = c.Executions.List(ctx, nil)
	assert.Error(t, err)
}

func TestRCServiceTriggers(t *testing.T) {
	s, c := rcNewServiceServer(t)
	ctx := context.Background()

	s.respond(http.StatusOK, `{"data":[{"id":"t1","type":"cron"}],"total":1}`)
	list, total, err := c.Triggers.List(ctx, &ServiceTriggerListOptions{Limit: 1, WorkflowID: "w1", Type: "cron"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, models.TriggerTypeCron, list[0].Type)
	assert.Equal(t, "limit=1&offset=0&workflow_id=w1&type=cron", s.last().RawQuery)

	s.respond(http.StatusCreated, `{"id":"t2"}`)
	tr, err := c.Triggers.Create(ctx, &ServiceCreateTriggerRequest{WorkflowID: "w1", Name: "n", Type: "manual"})
	require.NoError(t, err)
	assert.Equal(t, "t2", tr.ID)
	assert.Equal(t, false, s.last().Body["enabled"])

	enabled := true
	s.respond(http.StatusOK, `{"id":"t2","enabled":true}`)
	tr, err = c.Triggers.Update(ctx, "t2", &ServiceUpdateTriggerRequest{Enabled: &enabled})
	require.NoError(t, err)
	assert.True(t, tr.Enabled)
	assert.Equal(t, "PUT /api/v1/service/triggers/t2", s.last().Method+" "+s.last().Path)

	require.NoError(t, c.Triggers.Delete(ctx, "t2"))

	s.respond(http.StatusBadRequest, `{"code":"BAD","message":"bad"}`)
	_, _, err = c.Triggers.List(ctx, nil)
	assert.Error(t, err)
	_, err = c.Triggers.Create(ctx, &ServiceCreateTriggerRequest{})
	assert.Error(t, err)
	_, err = c.Triggers.Update(ctx, "t2", &ServiceUpdateTriggerRequest{})
	assert.Error(t, err)
	assert.Error(t, c.Triggers.Delete(ctx, "t2"))
}

func TestRCServiceCredentials(t *testing.T) {
	s, c := rcNewServiceServer(t)
	ctx := context.Background()

	s.respond(http.StatusOK, `{"credentials":[{"id":"c1","provider":"openai"}]}`)
	list, total, err := c.Credentials.List(ctx, &ServiceCredentialListOptions{Limit: 3, UserID: "u1", Provider: "openai"})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, 0, total)
	assert.Equal(t, "openai", list[0].Provider)
	assert.Equal(t, "limit=3&offset=0&user_id=u1&provider=openai", s.last().RawQuery)

	s.respond(http.StatusCreated, `{"id":"c2","name":"key"}`)
	cred, err := c.Credentials.Create(ctx, &ServiceCreateCredentialRequest{Name: "key", Provider: "openai", AuthType: "api_key", Data: map[string]string{"k": "v"}})
	require.NoError(t, err)
	assert.Equal(t, "c2", cred.ID)

	s.respond(http.StatusOK, `{"id":"c2","name":"renamed"}`)
	cred, err = c.Credentials.Update(ctx, "c2", &ServiceUpdateCredentialRequest{Name: "renamed"})
	require.NoError(t, err)
	assert.Equal(t, "renamed", cred.Name)

	require.NoError(t, c.Credentials.Delete(ctx, "c2"))
	assert.Equal(t, "DELETE /api/v1/service/credentials/c2", s.last().Method+" "+s.last().Path)

	s.respond(http.StatusForbidden, `{"code":"FORBIDDEN","message":"no"}`)
	_, _, err = c.Credentials.List(ctx, nil)
	assert.Error(t, err)
	_, err = c.Credentials.Create(ctx, &ServiceCreateCredentialRequest{})
	assert.Error(t, err)
	_, err = c.Credentials.Update(ctx, "c2", &ServiceUpdateCredentialRequest{})
	assert.Error(t, err)
	assert.Error(t, c.Credentials.Delete(ctx, "c2"))
}

func TestRCServiceClient_TransportErrors(t *testing.T) {
	c, err := NewServiceClient(ServiceClientConfig{Endpoint: rcClosedURL(t), SystemKey: "k"})
	require.NoError(t, err)
	ctx := context.Background()

	_, _, err = c.Workflows.List(ctx, nil)
	assert.ErrorContains(t, err, "request failed")
	_, err = c.Workflows.Get(ctx, "x")
	assert.Error(t, err)
	_, err = c.Workflows.Create(ctx, &ServiceCreateWorkflowRequest{})
	assert.Error(t, err)
	_, err = c.Workflows.Update(ctx, "x", &ServiceUpdateWorkflowRequest{})
	assert.Error(t, err)
	assert.Error(t, c.Workflows.Delete(ctx, "x"))

	_, _, err = c.Executions.List(ctx, nil)
	assert.Error(t, err)
	_, err = c.Executions.Get(ctx, "x")
	assert.Error(t, err)
	_, err = c.Executions.Start(ctx, "x", nil)
	assert.Error(t, err)
	assert.Error(t, c.Executions.Cancel(ctx, "x"))
	assert.Error(t, c.Executions.Retry(ctx, "x"))

	_, _, err = c.Triggers.List(ctx, nil)
	assert.Error(t, err)
	_, err = c.Triggers.Create(ctx, &ServiceCreateTriggerRequest{})
	assert.Error(t, err)
	_, err = c.Triggers.Update(ctx, "x", &ServiceUpdateTriggerRequest{})
	assert.Error(t, err)
	assert.Error(t, c.Triggers.Delete(ctx, "x"))

	_, _, err = c.Credentials.List(ctx, nil)
	assert.Error(t, err)
	_, err = c.Credentials.Create(ctx, &ServiceCreateCredentialRequest{})
	assert.Error(t, err)
	_, err = c.Credentials.Update(ctx, "x", &ServiceUpdateCredentialRequest{})
	assert.Error(t, err)
	assert.Error(t, c.Credentials.Delete(ctx, "x"))

	// Unmarshalable body and invalid URL.
	_, err = c.doRequest(ctx, http.MethodPost, "/x", map[string]any{"c": make(chan int)})
	assert.ErrorContains(t, err, "failed to marshal request body")
	c.config.Endpoint = "http://bad host"
	_, err = c.doRequest(ctx, http.MethodGet, "/x", nil)
	assert.ErrorContains(t, err, "failed to create request")
}
