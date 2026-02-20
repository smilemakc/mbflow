package storage

import (
	"os"
	"testing"

	"github.com/smilemakc/mbflow/go/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithEmbeddedDB(m))
}
