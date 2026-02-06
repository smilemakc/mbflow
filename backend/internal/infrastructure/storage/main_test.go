package storage

import (
	"os"
	"testing"

	"github.com/smilemakc/mbflow/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithEmbeddedDB(m))
}
