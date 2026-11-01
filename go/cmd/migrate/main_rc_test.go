package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithEmbeddedDB(m))
}

func rcEnv(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

type rcFakeMigrator struct {
	calls   []string
	failOn  string
	failErr error
}

func (f *rcFakeMigrator) do(name string) error {
	f.calls = append(f.calls, name)
	if name == f.failOn {
		return f.failErr
	}
	return nil
}

func (f *rcFakeMigrator) Init(context.Context) error   { return f.do("init") }
func (f *rcFakeMigrator) Up(context.Context) error     { return f.do("up") }
func (f *rcFakeMigrator) Down(context.Context) error   { return f.do("down") }
func (f *rcFakeMigrator) Status(context.Context) error { return f.do("status") }
func (f *rcFakeMigrator) Reset(context.Context) error  { return f.do("reset") }

func TestRCExecuteCommand(t *testing.T) {
	ctx := context.Background()
	for cmd, want := range map[string][]string{
		"init":   {"init"},
		"up":     {"init", "up"},
		"down":   {"down"},
		"status": {"status"},
		"reset":  {"reset"},
	} {
		f := &rcFakeMigrator{}
		require.NoError(t, executeCommand(ctx, f, cmd), cmd)
		assert.Equal(t, want, f.calls, cmd)
	}

	f := &rcFakeMigrator{failOn: "init", failErr: errors.New("boom")}
	err := executeCommand(ctx, f, "up")
	assert.EqualError(t, err, "init failed: boom")
	assert.Equal(t, []string{"init"}, f.calls, "up must not run after a failed init")

	err = executeCommand(ctx, &rcFakeMigrator{}, "sideways")
	assert.ErrorContains(t, err, "unknown command: sideways")
}

func TestRCRun_Errors(t *testing.T) {
	var out bytes.Buffer
	err := run(nil, rcEnv(nil), &out)
	assert.ErrorContains(t, err, "MBFLOW_DATABASE_URL (or DATABASE_URL) is required")
	assert.Contains(t, out.String(), "is required")

	out.Reset()
	err = run([]string{"-bogus"}, rcEnv(nil), &out)
	assert.Error(t, err)
	assert.Contains(t, out.String(), "flag provided but not defined")

	// Nothing listens on this port.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	out.Reset()
	err = run([]string{"-database-url", "postgres://u:p@" + addr + "/db?sslmode=disable"}, rcEnv(nil), &out)
	assert.Error(t, err)
	assert.Contains(t, out.String(), "failed to connect to database")
}

func TestRCRun_FullCycle(t *testing.T) {
	dsn := testutil.SetupTestDSN(t)
	var out bytes.Buffer

	// The flag wins over the environment; MBFLOW_DATABASE_URL over DATABASE_URL.
	env := rcEnv(map[string]string{"MBFLOW_DATABASE_URL": dsn, "DATABASE_URL": "postgres://invalid:1/x"})

	require.NoError(t, run([]string{"-command", "status"}, env, &out), out.String())
	assert.Contains(t, out.String(), "status=applied")
	assert.NotContains(t, out.String(), "status=pending")

	// Rolling back and re-applying exercises every down and up migration.
	out.Reset()
	require.NoError(t, run([]string{"-command", "reset", "-database-url", dsn}, rcEnv(nil), &out), out.String())
	assert.Contains(t, out.String(), "all migrations rolled back")

	out.Reset()
	require.NoError(t, run([]string{"-command", "status"}, rcEnv(map[string]string{"DATABASE_URL": dsn}), &out))
	assert.NotContains(t, out.String(), "status=applied")

	out.Reset()
	require.NoError(t, run([]string{"-command", "up"}, env, &out), out.String())
	assert.Contains(t, out.String(), "migrations applied successfully")

	out.Reset()
	require.NoError(t, run([]string{"-command", "up"}, env, &out), out.String())
	assert.Contains(t, out.String(), "no new migrations to run")

	out.Reset()
	require.NoError(t, run([]string{"-command", "down"}, env, &out), out.String())
	assert.Contains(t, out.String(), "migration rolled back successfully")

	out.Reset()
	require.NoError(t, run([]string{"-command", "down"}, env, &out), out.String())
	assert.Contains(t, out.String(), "no migrations to rollback")

	out.Reset()
	require.NoError(t, run([]string{"-command", "init"}, env, &out), out.String())

	out.Reset()
	err := run([]string{"-command", "nope"}, env, &out)
	assert.Error(t, err)
	assert.True(t, strings.Contains(out.String(), "migration command failed"))
}

func TestRCMain_Success(t *testing.T) {
	dsn := testutil.SetupTestDSN(t)
	oldArgs, oldStdout := os.Args, os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	defer devNull.Close()
	os.Args = []string{"migrate", "-command", "status", "-database-url", dsn}
	os.Stdout = devNull
	defer func() { os.Args, os.Stdout = oldArgs, oldStdout }()

	main() // returns normally on success (os.Exit only on failure)
}
