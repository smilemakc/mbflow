package migrations

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rcMigrationName = regexp.MustCompile(`^(\d{3})_([a-z0-9_]+)\.(up|down)\.sql$`)

// TestRCMigrationFiles checks that every embedded migration has both an up
// and a down script, that versions are unique and contiguous, and that no
// script is empty. (Applying and rolling back all of them against PostgreSQL
// is covered by cmd/migrate tests.)
func TestRCMigrationFiles(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	require.NoError(t, err)

	type pair struct{ name, up, down string }
	byVersion := map[string]*pair{}
	for _, e := range entries {
		m := rcMigrationName.FindStringSubmatch(e.Name())
		require.NotNil(t, m, "unexpected file name %q", e.Name())
		p := byVersion[m[1]]
		if p == nil {
			p = &pair{name: m[2]}
			byVersion[m[1]] = p
		}
		assert.Equal(t, p.name, m[2], "version %s used by two migrations", m[1])

		data, err := fs.ReadFile(FS, e.Name())
		require.NoError(t, err)
		assert.NotEmpty(t, strings.TrimSpace(string(data)), "%s is empty", e.Name())
		if m[3] == "up" {
			p.up = e.Name()
		} else {
			p.down = e.Name()
		}
	}

	require.NotEmpty(t, byVersion)
	for i := 1; i <= len(byVersion); i++ {
		v := fmt.Sprintf("%03d", i)
		p, ok := byVersion[v]
		if !assert.True(t, ok, "missing migration version %s", v) {
			continue
		}
		assert.NotEmpty(t, p.up, "%s has no up script", v)
		assert.NotEmpty(t, p.down, "%s has no down script", v)
	}
}
