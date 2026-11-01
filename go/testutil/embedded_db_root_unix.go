//go:build unix

package testutil

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// PostgreSQL refuses to run initdb and the server as root, so tests run as
// root (containers, CI images) start them as an unprivileged user. The
// embedded-postgres library executes <binaries>/bin/initdb and pg_ctl
// directly; under root those two are wrapper scripts that run this test
// binary again in helper mode, and the helper starts the real program with
// the unprivileged user's credentials.
const (
	runAsEnv      = "MBFLOW_TESTUTIL_RUN_AS"      // "uid:gid"
	runAsProgEnv  = "MBFLOW_TESTUTIL_RUN_AS_PROG" // real initdb/pg_ctl
	runAsOwnedEnv = "MBFLOW_TESTUTIL_RUN_AS_OWN"  // directory handed to the user first
)

func init() {
	if os.Getenv(runAsEnv) == "" {
		return
	}
	os.Exit(runAsHelper())
}

// runAsHelper runs the program named in the environment with the arguments of
// this process, as the user named in the environment, and returns its exit code.
func runAsHelper() int {
	uid, gid, err := parseIDs(os.Getenv(runAsEnv))
	if err != nil {
		fmt.Fprintln(os.Stderr, "testutil helper:", err)
		return 2
	}
	if dir := os.Getenv(runAsOwnedEnv); dir != "" {
		if err := chownTree(dir, uid, gid); err != nil {
			fmt.Fprintln(os.Stderr, "testutil helper:", err)
			return 2
		}
	}

	cmd := exec.Command(os.Getenv(runAsProgEnv), os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Dir = os.TempDir()
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: gid}}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "MBFLOW_TESTUTIL_") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "testutil helper:", err)
		return 2
	}
	return 0
}

func parseIDs(s string) (uid, gid uint32, err error) {
	u, g, ok := strings.Cut(s, ":")
	if !ok {
		return 0, 0, fmt.Errorf("invalid %s %q", runAsEnv, s)
	}
	uv, err := strconv.ParseUint(u, 10, 32)
	if err != nil {
		return 0, 0, err
	}
	gv, err := strconv.ParseUint(g, 10, 32)
	if err != nil {
		return 0, 0, err
	}
	return uint32(uv), uint32(gv), nil
}

func chownTree(dir string, uid, gid uint32) error {
	return filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Lchown(path, int(uid), int(gid))
	})
}

// unprivilegedUser returns the account PostgreSQL runs as under root.
func unprivilegedUser() (uid, gid uint32, err error) {
	for _, name := range []string{"postgres", "nobody"} {
		u, lookupErr := user.Lookup(name)
		if lookupErr != nil {
			continue
		}
		return parseIDs(u.Uid + ":" + u.Gid)
	}
	return 0, 0, errors.New("running as root and no unprivileged user (postgres, nobody) to run PostgreSQL as")
}

// prepareRootRun is called when the tests run as root. It extracts the
// PostgreSQL binaries to binDir and returns the directory to use as the
// library's binaries path, whose initdb and pg_ctl run as an unprivileged
// user. runtimeDir is handed to that user before each call.
func prepareRootRun(port uint32, runtimeDir, binDir string) (string, error) {
	uid, gid, err := unprivilegedUser()
	if err != nil {
		return "", err
	}

	// Let the library extract the binaries; its Start then fails at initdb
	// (root), which is expected here.
	extractDir := runtimeDir + "-extract"
	defer func() { _ = os.RemoveAll(extractDir) }()
	_ = embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(port).
		BinariesPath(binDir).
		RuntimePath(extractDir).
		Logger(&bytes.Buffer{})).Start()
	if _, err := os.Stat(filepath.Join(binDir, "bin", "pg_ctl")); err != nil {
		return "", fmt.Errorf("extract postgres binaries: %w", err)
	}
	if err := chownTree(binDir, uid, gid); err != nil {
		return "", err
	}

	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	wrapDir := runtimeDir + "-wrap"
	if err := os.MkdirAll(filepath.Join(wrapDir, "bin"), 0o755); err != nil {
		return "", err
	}
	for _, prog := range []string{"initdb", "pg_ctl"} {
		script := fmt.Sprintf("#!/bin/sh\nexec env %s=%d:%d %s=%s %s=%s %s \"$@\"\n",
			runAsEnv, uid, gid,
			runAsProgEnv, shellQuote(filepath.Join(binDir, "bin", prog)),
			runAsOwnedEnv, shellQuote(runtimeDir),
			shellQuote(self))
		if err := os.WriteFile(filepath.Join(wrapDir, "bin", prog), []byte(script), 0o755); err != nil { //nolint:gosec // executable test wrapper
			return "", err
		}
	}
	return wrapDir, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
