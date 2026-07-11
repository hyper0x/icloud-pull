package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// These are e2e tests that build icloud-pull as a real subprocess,
// similar to cmdguard's e2e_test.go pattern.

var (
	binPath  string
	binOnce  sync.Once
	errBuild error // test helper, not a sentinel
)

// buildOnce compiles icloud-pull into a temp binary, reused across tests.
func buildOnce(t *testing.T) string {
	t.Helper()
	binOnce.Do(func() {
		dir, err := os.MkdirTemp("", "icloud-pull-e2e-*")
		if err != nil {
			errBuild = err
			return
		}
		binPath = filepath.Join(dir, "icloud-pull")
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			errBuild = err
		}
	})
	if errBuild != nil {
		t.Fatalf("build icloud-pull: %v", errBuild)
	}
	return binPath
}

// runBin executes the icloud-pull binary with given args and returns
// stdout, stderr, and exit code.
func runBin(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	bin := buildOnce(t)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("exec: %v", err)
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

// TestNoArgs prints usage and exits 1.
func TestNoArgs(t *testing.T) {
	stdout, _, code := runBin(t)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Usage") {
		t.Error("expected usage message in stdout")
	}
}

// TestHelpFlag prints help and exits 0.
func TestHelpFlag(t *testing.T) {
	for _, flag := range []string{"--help", "-h", "help"} {
		t.Run(flag, func(t *testing.T) {
			stdout, _, code := runBin(t, flag)
			if code != 0 {
				t.Errorf("expected exit 0, got %d", code)
			}
			if !strings.Contains(stdout, "icloud-pull") {
				t.Error("expected banner in stdout")
			}
		})
	}
}

// TestVersionFlag prints version and exits 0.
func TestVersionFlag(t *testing.T) {
	for _, flag := range []string{"--version", "-v", "version"} {
		t.Run(flag, func(t *testing.T) {
			stdout, _, code := runBin(t, flag)
			if code != 0 {
				t.Errorf("expected exit 0, got %d", code)
			}
			if !strings.Contains(stdout, "icloud-pull") {
				t.Error("expected 'icloud-pull' in version output")
			}
		})
	}
}

// TestStatusCommand verifies the status subcommand works on a temp dir.
func TestStatusCommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := runBin(t, "status", dir)
	if code != 0 {
		t.Errorf("expected exit 0, got %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Scanned") {
		t.Errorf("expected scan report in stdout, got: %s", stdout)
	}
	if !strings.Contains(stdout, "0 evicted") {
		t.Errorf("expected 0 evicted in a temp dir, got: %s", stdout)
	}
}

// TestStatusJSON verifies --json output is valid JSON.
func TestStatusJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := runBin(t, "status", "--json", dir)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, stdout)
	}
	if out["root"] == nil {
		t.Error("expected 'root' key in JSON")
	}
	if out["total"].(float64) != 1 {
		t.Errorf("expected total=1, got %v", out["total"])
	}
	if out["evicted"].(float64) != 0 {
		t.Errorf("expected evicted=0, got %v", out["evicted"])
	}
}

// TestDownloadNoEvicted verifies download on a dir with no evicted files.
func TestDownloadNoEvicted(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := runBin(t, "download", dir)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "nothing evicted") {
		t.Errorf("expected 'nothing evicted' message, got: %s", stdout)
	}
}

// TestDownloadJSON verifies download --json on a dir with no evicted files.
func TestDownloadJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := runBin(t, "download", "--json", dir)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, stdout)
	}
}

// TestUnknownCommand exits with error.
func TestUnknownCommand(t *testing.T) {
	_, stderr, code := runBin(t, "bogus")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("expected 'unknown command' in stderr, got: %s", stderr)
	}
}

// TestStatusNonexistentPath exits with error.
func TestStatusNonexistentPath(t *testing.T) {
	_, stderr, code := runBin(t, "status", "/nonexistent/path/abc123")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "error") {
		t.Errorf("expected error in stderr, got: %s", stderr)
	}
}

// TestMultiplePaths scans multiple directories in one invocation.
func TestMultiplePaths(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir1, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "b.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := runBin(t, "status", dir1, dir2)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	// Should have two scan reports.
	if !strings.Contains(stdout, "Scanned") {
		t.Errorf("expected scanning output, got: %s", stdout)
	}
}
