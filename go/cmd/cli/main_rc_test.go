package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/config"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithEmbeddedDB(m))
}

const rcPassword = "Str0ng!Passw0rd"

type rcHarness struct {
	cli    *cli
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	env    map[string]string
}

func rcNewHarness(stdin string) *rcHarness {
	h := &rcHarness{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}, env: map[string]string{}}
	h.cli = &cli{
		stdout: h.stdout,
		stderr: h.stderr,
		stdin:  bufio.NewReader(strings.NewReader(stdin)),
		getenv: func(k string) string { return h.env[k] },
		newGatewayProvider: func(*config.AuthConfig) (gatewayProvider, error) {
			return nil, errors.New("no gateway in tests")
		},
	}
	return h
}

func (h *rcHarness) run(args ...string) int {
	h.stdout.Reset()
	h.stderr.Reset()
	return h.cli.run(args)
}

// rcAPI serves the workflow endpoints used by the CLI.
func rcAPI(t *testing.T, workflows []*pkgmodels.Workflow) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.Header.Get("Authorization") != "Bearer cli-key":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":"UNAUTHORIZED","message":"bad key"}`))
		case r.URL.Path == "/api/v1/workflows":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": workflows, "total": len(workflows)})
		case strings.HasPrefix(r.URL.Path, "/api/v1/workflows/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/v1/workflows/")
			for _, wf := range workflows {
				if wf.ID == id {
					_ = json.NewEncoder(w).Encode(wf)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"WORKFLOW_NOT_FOUND","message":"workflow not found"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func rcWorkflow() *pkgmodels.Workflow {
	return &pkgmodels.Workflow{
		ID: "wf-1", Name: "Demo", Description: "demo flow", Status: pkgmodels.WorkflowStatusActive, Tags: []string{"x", "y"},
		Nodes: []*pkgmodels.Node{
			{ID: "fetch", Name: "Fetch", Type: "http", Config: map[string]any{"method": "GET", "url": "https://example.com"}},
			{ID: "shape", Name: "Shape", Type: "transform"},
		},
		Edges: []*pkgmodels.Edge{{ID: "e1", From: "fetch", To: "shape"}},
	}
}

func TestRCDispatch_Basics(t *testing.T) {
	h := rcNewHarness("")

	assert.Equal(t, 0, h.run("version"))
	assert.Equal(t, "MBFlow CLI v"+version+"\n", h.stdout.String())

	for _, arg := range []string{"help", "-h", "--help"} {
		assert.Equal(t, 0, h.run(arg))
		assert.Contains(t, h.stdout.String(), "USAGE:")
	}

	assert.Equal(t, 1, h.run())
	assert.Contains(t, h.stderr.String(), "Error: a command is required")
	assert.Contains(t, h.stderr.String(), "USAGE:")

	assert.Equal(t, 1, h.run("frobnicate"))
	assert.Contains(t, h.stderr.String(), "unknown command: frobnicate")

	assert.Equal(t, 1, h.run("workflow"))
	assert.Contains(t, h.stderr.String(), "workflow command requires a subcommand (list, show)")
	assert.Equal(t, 1, h.run("workflow", "delete"))
	assert.Contains(t, h.stderr.String(), "unknown workflow subcommand: delete")
	assert.NotContains(t, h.stderr.String(), "USAGE:")

	for _, cmd := range []string{"user", "admin", "system-key"} {
		assert.Equal(t, 1, h.run(cmd), cmd)
		assert.Contains(t, h.stderr.String(), cmd+" command requires a subcommand (create)")
		assert.Equal(t, 1, h.run(cmd, "nope"), cmd)
	}

	// -h on a subcommand prints flag help and succeeds.
	assert.Equal(t, 0, h.run("workflow", "list", "-h"))
	assert.Contains(t, h.stderr.String(), "-endpoint")
	assert.Equal(t, 1, h.run("workflow", "list", "-unknown"))
}

func TestRCWorkflowList(t *testing.T) {
	srv := rcAPI(t, []*pkgmodels.Workflow{rcWorkflow(), {ID: "wf-2", Name: "Bare", Status: pkgmodels.WorkflowStatusDraft}})
	h := rcNewHarness("")
	h.env["MBFLOW_ENDPOINT"] = srv.URL
	h.env["MBFLOW_API_KEY"] = "cli-key"

	require.Equal(t, 0, h.run("workflow", "list"), h.stderr.String())
	out := h.stdout.String()
	assert.Contains(t, out, "Found 2 workflow(s):")
	assert.Contains(t, out, "ID:          wf-1")
	assert.Contains(t, out, "Description: demo flow")
	assert.Contains(t, out, "Nodes:       2")
	assert.Contains(t, out, "Tags:        x, y")
	assert.Equal(t, 2, strings.Count(out, "---"))

	// Flags override the environment.
	assert.Equal(t, 1, h.run("workflow", "list", "-api-key", "wrong"))
	assert.Contains(t, h.stderr.String(), "failed to list workflows")

	empty := rcAPI(t, nil)
	require.Equal(t, 0, h.run("workflow", "list", "-endpoint", empty.URL, "-timeout", "5s"))
	assert.Equal(t, "No workflows found\n", h.stdout.String())

	assert.Equal(t, 1, h.run("workflow", "list", "-endpoint", rcClosedURL(t)))
	assert.Contains(t, h.stderr.String(), "failed to create client")
}

func TestRCWorkflowShow(t *testing.T) {
	srv := rcAPI(t, []*pkgmodels.Workflow{rcWorkflow()})
	h := rcNewHarness("")
	h.env["MBFLOW_API_KEY"] = "cli-key"
	ep := []string{"-endpoint", srv.URL}

	require.Equal(t, 0, h.run(append([]string{"workflow", "show", "wf-1"}, ep...)...), h.stderr.String())
	assert.Contains(t, h.stdout.String(), "flowchart TB")
	assert.Contains(t, h.stdout.String(), "fetch --> shape")

	require.Equal(t, 0, h.run(append([]string{"workflow", "show", "wf-1", "-format", "ASCII", "-compact", "-color=false"}, ep...)...))
	assert.Contains(t, h.stdout.String(), "Demo")
	assert.Contains(t, h.stdout.String(), "fetch (http)")
	assert.NotContains(t, h.stdout.String(), "\033[")

	file := filepath.Join(t.TempDir(), "d.mmd")
	require.Equal(t, 0, h.run(append([]string{"workflow", "show", "wf-1", "-direction", "LR", "-output", file}, ep...)...))
	assert.Equal(t, "Diagram saved to "+file+"\n", h.stdout.String())
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(t, string(data), "flowchart LR")

	assert.Equal(t, 1, h.run(append([]string{"workflow", "show", "wf-1", "-output", filepath.Join(t.TempDir(), "missing", "x.mmd")}, ep...)...))
	assert.Contains(t, h.stderr.String(), "failed to write to file")

	assert.Equal(t, 1, h.run(append([]string{"workflow", "show", "missing"}, ep...)...))
	assert.Contains(t, h.stderr.String(), "failed to get workflow 'missing'")
	assert.Contains(t, h.stderr.String(), "WORKFLOW_NOT_FOUND")

	assert.Equal(t, 1, h.run("workflow", "show", "wf-1", "-format", "svg"))
	assert.Contains(t, h.stderr.String(), "invalid format 'svg'")

	assert.Equal(t, 1, h.run("workflow", "show"))
	assert.Contains(t, h.stderr.String(), "workflow show requires a workflow ID")
	assert.Equal(t, 1, h.run("workflow", "show", "-format", "ascii"))
	assert.Contains(t, h.stderr.String(), "workflow show requires a workflow ID")

	assert.Equal(t, 1, h.run("workflow", "show", "wf-1", "-bogus"))
	assert.Equal(t, 1, h.run("workflow", "show", "wf-1", "-endpoint", rcClosedURL(t)))
	assert.Contains(t, h.stderr.String(), "failed to create client")
}

func rcClosedURL(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return "http://" + addr
}

func TestRCPromptPassword(t *testing.T) {
	// Both lines are read through one reader (regression: a fresh
	// bufio.Reader per prompt swallowed the confirmation line).
	h := rcNewHarness("first\nsecond\n")
	assert.Equal(t, "first", h.cli.promptPassword("p1: "))
	assert.Equal(t, "second", h.cli.promptPassword("p2: "))
	assert.Equal(t, "p1: p2: ", h.stdout.String())

	h = rcNewHarness("fallback\n")
	h.cli.readSecret = func() (string, bool, error) { return "hidden", true, nil }
	assert.Equal(t, "hidden", h.cli.promptPassword("pw: "))
	assert.Equal(t, "pw: \n", h.stdout.String())

	h.cli.readSecret = func() (string, bool, error) { return "", true, errors.New("tty gone") }
	assert.Equal(t, "fallback", h.cli.promptPassword("pw: "))

	h.cli.readSecret = func() (string, bool, error) { return "", false, nil }
	assert.Equal(t, "", h.cli.promptPassword("pw: "))

	pw := ""
	h = rcNewHarness("same\nother\n")
	assert.EqualError(t, h.cli.requirePassword(&pw, "pw: "), "passwords do not match")
	pw = ""
	h = rcNewHarness("\n")
	assert.EqualError(t, h.cli.requirePassword(&pw, "pw: "), "password cannot be empty")
	pw = ""
	h = rcNewHarness("same\nsame\n")
	require.NoError(t, h.cli.requirePassword(&pw, "pw: "))
	assert.Equal(t, "same", pw)
}

func TestRCUserCreate_Validation(t *testing.T) {
	h := rcNewHarness("")
	assert.Equal(t, 1, h.run("user", "create"))
	assert.Contains(t, h.stderr.String(), "-email is required")
	assert.Equal(t, 1, h.run("user", "create", "-email", "a@b.c"))
	assert.Contains(t, h.stderr.String(), "-username is required")
	assert.Equal(t, 1, h.run("user", "create", "-email", "a@b.c", "-username", "u"))
	assert.Contains(t, h.stderr.String(), "must specify either -gateway or -local")
	assert.Equal(t, 1, h.run("user", "create", "-email", "a@b.c", "-username", "u", "-local"))
	assert.Contains(t, h.stderr.String(), "password cannot be empty")
	assert.Equal(t, 1, h.run("user", "create", "-email", "a@b.c", "-username", "u", "-local", "-password", rcPassword))
	assert.Contains(t, h.stderr.String(), "DATABASE_URL environment variable is required for -local mode")
	assert.Equal(t, 1, h.run("user", "create", "-nope"))

	h.env["DATABASE_URL"] = "postgres://u:p@" + strings.TrimPrefix(rcClosedURL(t), "http://") + "/x?sslmode=disable"
	assert.Equal(t, 1, h.run("user", "create", "-email", "a@b.c", "-username", "u", "-local", "-password", rcPassword))
	assert.Contains(t, h.stderr.String(), "failed to connect to database")

	assert.Equal(t, 1, h.run("admin", "create"))
	assert.Contains(t, h.stderr.String(), "-email is required")
	assert.Equal(t, 1, h.run("admin", "create", "-email", "a@b.c"))
	assert.Contains(t, h.stderr.String(), "-username is required")
	assert.Equal(t, 1, h.run("admin", "create", "-email", "a@b.c", "-username", "u"))
	assert.Contains(t, h.stderr.String(), "password cannot be empty")
	assert.Equal(t, 1, h.run("admin", "create", "-bogus"))
	delete(h.env, "DATABASE_URL")
	assert.Equal(t, 1, h.run("admin", "create", "-email", "a@b.c", "-username", "u", "-password", rcPassword))
	assert.Contains(t, h.stderr.String(), "DATABASE_URL environment variable is required\n")

	assert.Equal(t, 1, h.run("system-key", "create"))
	assert.Contains(t, h.stderr.String(), "-name is required")
	assert.Equal(t, 1, h.run("system-key", "create", "-name", "k"))
	assert.Contains(t, h.stderr.String(), "DATABASE_URL")
	assert.Equal(t, 1, h.run("system-key", "create", "-bogus"))
	h.env["DATABASE_URL"] = "postgres://u:p@" + strings.TrimPrefix(rcClosedURL(t), "http://") + "/x?sslmode=disable"
	assert.Equal(t, 1, h.run("system-key", "create", "-name", "k"))
	assert.Contains(t, h.stderr.String(), "failed to connect to database")
}

type rcGateway struct {
	available bool
	result    *auth.ProviderAuthResult
	err       error
	req       *auth.CreateUserRequest
	closed    bool
}

func (g *rcGateway) IsAvailable() bool { return g.available }
func (g *rcGateway) CreateUser(_ context.Context, req *auth.CreateUserRequest) (*auth.ProviderAuthResult, error) {
	g.req = req
	return g.result, g.err
}
func (g *rcGateway) Close() error { g.closed = true; return nil }

func TestRCUserCreate_Gateway(t *testing.T) {
	h := rcNewHarness("")
	args := []string{"user", "create", "-email", "g@x.io", "-username", "gu", "-password", rcPassword, "-gateway", "-full-name", "G U", "-phone", "+1"}

	assert.Equal(t, 1, h.run(args...))
	assert.Contains(t, h.stderr.String(), "MBFLOW_AUTH_GRPC_ADDRESS environment variable is required")

	h.env["MBFLOW_AUTH_GRPC_ADDRESS"] = "gw:50051"
	assert.Equal(t, 1, h.run(args...))
	assert.Contains(t, h.stderr.String(), "failed to create gRPC provider: no gateway in tests")

	gw := &rcGateway{}
	var gotCfg *config.AuthConfig
	h.cli.newGatewayProvider = func(cfg *config.AuthConfig) (gatewayProvider, error) { gotCfg = cfg; return gw, nil }
	assert.Equal(t, 1, h.run(args...))
	assert.Contains(t, h.stderr.String(), "gRPC auth provider is not available")
	assert.True(t, gw.closed)
	assert.Equal(t, "gw:50051", gotCfg.GRPCAddress)

	gw.available = true
	gw.err = errors.New("email taken")
	assert.Equal(t, 1, h.run(args...))
	assert.Contains(t, h.stderr.String(), "failed to create user via auth-gateway: email taken")

	gw.err = nil
	assert.Equal(t, 1, h.run(args...))
	assert.Contains(t, h.stderr.String(), "auth-gateway returned no user")

	gw.result = &auth.ProviderAuthResult{
		User:        &pkgmodels.User{ID: "u-1", Email: "g@x.io", Username: "gu", FullName: "G U"},
		AccessToken: "abcdefghijklmnopqrstuvwxyz",
	}
	require.Equal(t, 0, h.run(args...), h.stderr.String())
	out := h.stdout.String()
	assert.Contains(t, out, "User created successfully via auth-gateway!")
	assert.Contains(t, out, "Name:     G U")
	assert.Contains(t, out, "Token:    abcdefghijklmnopqrst...")
	assert.Equal(t, &auth.CreateUserRequest{Email: "g@x.io", Phone: "+1", Username: "gu", Password: rcPassword, FullName: "G U", AccountType: "human"}, gw.req)

	gw.result.AccessToken = "short"
	require.Equal(t, 0, h.run(args...))
	assert.Contains(t, h.stdout.String(), "Token:    short...")
}

func TestRCLocalUsersAndSystemKeys(t *testing.T) {
	dsn := testutil.SetupTestDSN(t)
	h := rcNewHarness("")
	h.env["DATABASE_URL"] = dsn

	// No admin yet: system key creation needs a creator.
	assert.Equal(t, 1, h.run("system-key", "create", "-name", "k0"))
	assert.Contains(t, h.stderr.String(), "no admin user found")

	// Password prompted twice from stdin.
	h.cli.stdin = bufio.NewReader(strings.NewReader(rcPassword + "\n" + rcPassword + "\n"))
	require.Equal(t, 0, h.run("admin", "create", "-email", "admin@x.io", "-username", "admin", "-full-name", "Ad Min"), h.stderr.String())
	assert.Contains(t, h.stdout.String(), "Admin user created successfully!")
	assert.Contains(t, h.stdout.String(), "Name:     Ad Min")
	assert.Contains(t, h.stdout.String(), "Admin:    true")

	assert.Equal(t, 1, h.run("admin", "create", "-email", "admin@x.io", "-username", "other", "-password", rcPassword))
	assert.Contains(t, h.stderr.String(), "user with email 'admin@x.io' already exists")
	assert.Equal(t, 1, h.run("user", "create", "-email", "new@x.io", "-username", "admin", "-password", rcPassword, "-local"))
	assert.Contains(t, h.stderr.String(), "user with username 'admin' already exists")
	assert.Equal(t, 1, h.run("user", "create", "-email", "new@x.io", "-username", "new", "-password", "weak", "-local"))
	assert.Contains(t, h.stderr.String(), "minimum 8 characters")

	require.Equal(t, 0, h.run("user", "create", "-email", "u@x.io", "-username", "plain", "-password", rcPassword, "-local"), h.stderr.String())
	assert.Contains(t, h.stdout.String(), "User created successfully in local database!")
	assert.Contains(t, h.stdout.String(), "Admin:    false")
	assert.NotContains(t, h.stdout.String(), "Name:")

	require.Equal(t, 0, h.run("user", "create", "-email", "a2@x.io", "-username", "admin2", "-password", rcPassword, "-local", "-admin"), h.stderr.String())
	assert.Contains(t, h.stdout.String(), "Admin:    true")

	// System keys: default creator is the first admin; explicit creator by email.
	require.Equal(t, 0, h.run("system-key", "create", "-name", "k1", "-service", "backend"), h.stderr.String())
	out := h.stdout.String()
	assert.Contains(t, out, "System key created successfully!")
	assert.Contains(t, out, "Service: backend")
	assert.Contains(t, out, "sysk_")
	assert.NotContains(t, out, "Expires:")

	require.Equal(t, 0, h.run("system-key", "create", "-name", "k2", "-created-by", "u@x.io", "-expires", "30", "-description", "d"), h.stderr.String())
	assert.Contains(t, h.stdout.String(), "Expires:")

	assert.Equal(t, 1, h.run("system-key", "create", "-name", "k3", "-created-by", "ghost@x.io"))
	assert.Contains(t, h.stderr.String(), "user with email 'ghost@x.io' not found")

	assert.Equal(t, 1, h.run("system-key", "create", "-name", strings.Repeat("n", 300)))
	assert.Contains(t, h.stderr.String(), "failed to create system key")
}

type rcCreatorLookup struct {
	users []*storagemodels.UserModel
	err   error
}

func (r rcCreatorLookup) FindByEmail(context.Context, string) (*storagemodels.UserModel, error) {
	return nil, errors.New("not found")
}
func (r rcCreatorLookup) FindAll(context.Context, int, int) ([]*storagemodels.UserModel, error) {
	return r.users, r.err
}

func TestRCResolveCreatorID(t *testing.T) {
	ctx := context.Background()
	_, err := resolveCreatorID(ctx, rcCreatorLookup{err: errors.New("db down")}, "")
	assert.ErrorContains(t, err, "failed to query users: db down")

	id := uuid.New()
	got, err := resolveCreatorID(ctx, rcCreatorLookup{users: []*storagemodels.UserModel{{ID: uuid.New()}, {ID: id, IsAdmin: true}}}, "")
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestRCNewCLIAndMain(t *testing.T) {
	c := newCLI()
	assert.Equal(t, os.Stdout, c.stdout)
	// Under `go test` stdin is not a terminal.
	_, ok, err := c.readSecret()
	assert.False(t, ok)
	assert.NoError(t, err)
	p, err := c.newGatewayProvider(&config.AuthConfig{})
	require.NoError(t, err)
	assert.False(t, p.IsAvailable())
	assert.NoError(t, p.Close())

	oldArgs, oldExit, oldStdout := os.Args, exit, os.Stdout
	defer func() { os.Args, exit, os.Stdout = oldArgs, oldExit, oldStdout }()
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	defer devNull.Close()
	os.Stdout = devNull
	code := -1
	exit = func(c int) { code = c }
	os.Args = []string{"mbflow-cli", "version"}
	main()
	assert.Equal(t, 0, code)
}
