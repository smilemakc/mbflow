package grpcclient_test

import (
	"testing"

	"github.com/smilemakc/mbflow/go/sdk/internal"
	"github.com/smilemakc/mbflow/go/sdk/internal/grpcclient"
)

func TestGRPCTransport_New(t *testing.T) {
	tr, err := grpcclient.New("localhost:50051", &grpcclient.Config{
		SystemKey: "test-key",
		Insecure:  true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer tr.Close()

	var _ internal.Transport = tr
}
