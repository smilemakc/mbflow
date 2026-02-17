package mbflow_test

import (
	"testing"

	mbflow "github.com/smilemakc/mbflow/sdk/go"
)

func TestNewClient_HTTP(t *testing.T) {
	client, err := mbflow.NewClient(
		mbflow.WithHTTP("http://localhost:8585"),
		mbflow.WithAPIKey("test-key"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if client.Workflows() == nil {
		t.Error("Workflows() returned nil")
	}
	if client.Executions() == nil {
		t.Error("Executions() returned nil")
	}
	if client.Triggers() == nil {
		t.Error("Triggers() returned nil")
	}
	if client.Credentials() == nil {
		t.Error("Credentials() returned nil")
	}
}

func TestNewClient_NoTransport(t *testing.T) {
	_, err := mbflow.NewClient()
	if err == nil {
		t.Error("expected error when no transport specified")
	}
}
