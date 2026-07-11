package cli

import (
	"testing"
)

// TestRunNoArgs verifies that Run with no args prints usage and returns 1.
func TestRunNoArgs(t *testing.T) {
	// Run writes to os.Stdout/os.Stderr, so we can't easily capture it
	// in a unit test. Instead, we test the exit code.
	code := Run(nil)
	if code != 1 {
		t.Errorf("expected exit 1 for no args, got %d", code)
	}
}

// TestRunHelp verifies that help flags return exit 0.
func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"-h"},
		{"help"},
	} {
		t.Run(args[0], func(t *testing.T) {
			code := Run(args)
			if code != 0 {
				t.Errorf("expected exit 0 for %s, got %d", args[0], code)
			}
		})
	}
}

// TestRunVersion verifies that version flags return exit 0.
func TestRunVersion(t *testing.T) {
	for _, args := range [][]string{
		{"--version"},
		{"version"},
	} {
		t.Run(args[0], func(t *testing.T) {
			code := Run(args)
			if code != 0 {
				t.Errorf("expected exit 0 for %s, got %d", args[0], code)
			}
		})
	}
}

// TestRunUnknownCommand verifies that an unknown command returns exit 1.
func TestRunUnknownCommand(t *testing.T) {
	code := Run([]string{"bogus"})
	if code != 1 {
		t.Errorf("expected exit 1 for unknown command, got %d", code)
	}
}

// TestRunStatusNoPath verifies that status without a path returns exit 1.
func TestRunStatusNoPath(t *testing.T) {
	code := Run([]string{"status"})
	if code != 1 {
		t.Errorf("expected exit 1 for status with no path, got %d", code)
	}
}

// TestRunDownloadNoPath verifies that download without a path returns exit 1.
func TestRunDownloadNoPath(t *testing.T) {
	code := Run([]string{"download"})
	if code != 1 {
		t.Errorf("expected exit 1 for download with no path, got %d", code)
	}
}

// TestRunStatusNonexistentPath verifies error handling for bad paths.
func TestRunStatusNonexistentPath(t *testing.T) {
	code := Run([]string{"status", "/nonexistent/path/abc123"})
	if code != 1 {
		t.Errorf("expected exit 1 for nonexistent path, got %d", code)
	}
}
