package server

import (
	"os"
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/testutil"
)

func TestMain(m *testing.M) {
	// The default encryption service is initialized once per process, so the
	// key must be in place before the first server is created. With it the
	// credentials and rental-key features are enabled in every test server.
	if os.Getenv(crypto.KeyEnvVar) == "" {
		_ = os.Setenv(crypto.KeyEnvVar, "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=") // 32 bytes, test only
	}
	os.Exit(testutil.RunWithEmbeddedDB(m))
}
