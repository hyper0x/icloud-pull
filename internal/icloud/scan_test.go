package icloud

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestScanEmptyDir verifies that scanning an empty directory returns
// zero files and no error.
func TestScanEmptyDir(t *testing.T) {
	dir := t.TempDir()

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 files, got %d", result.Total)
	}
	if result.EvictedCount() != 0 {
		t.Errorf("expected 0 evicted, got %d", result.EvictedCount())
	}
}

// TestScanLocalFiles verifies that regular (non-evicted) files are
// classified as StatusLocal.
func TestScanLocalFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a few regular files.
	for _, name := range []string{"a.txt", "b.md", "c.json"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected 3 files, got %d", result.Total)
	}
	if result.EvictedCount() != 0 {
		t.Errorf("expected 0 evicted, got %d", result.EvictedCount())
	}
	if len(result.Local) != 3 {
		t.Errorf("expected 3 local, got %d", len(result.Local))
	}
}

// TestScanNestedDirs verifies that the walker descends into subdirectories.
func TestScanNestedDirs(t *testing.T) {
	dir := t.TempDir()

	subdir := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "top.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "deep.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 files, got %d", result.Total)
	}
}

// TestScanSkipsSymlinks verifies that symlinks are not counted.
func TestScanSkipsSymlinks(t *testing.T) {
	dir := t.TempDir()

	target := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(target, []byte("real"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	// Should count 1 (the real file), not 2.
	if result.Total != 1 {
		t.Errorf("expected 1 file (symlink skipped), got %d", result.Total)
	}
}

// TestScanNonexistentPath verifies that scanning a nonexistent path
// returns an error.
func TestScanNonexistentPath(t *testing.T) {
	_, err := Scan("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}

// TestScanNotADirectory verifies that scanning a file (not a directory)
// returns an error.
func TestScanNotADirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Scan(filePath)
	if err == nil {
		t.Error("expected error for non-directory path, got nil")
	}
}

// TestDownloadAlreadyLocal verifies that Download returns
// DownloadAlreadyLocal for a non-evicted file.
func TestDownloadAlreadyLocal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "local.txt")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := File{Path: path, Status: StatusLocal}
	dr := Download(f)
	if dr.Status != DownloadAlreadyLocal {
		t.Errorf("expected DownloadAlreadyLocal, got %d", dr.Status)
	}
}

// TestDownloadNonEvictedFile verifies that calling Download on a file
// with StatusEvicted but that is actually local on disk works correctly.
// (On macOS test temp dirs are not in iCloud, so we can't create real
// evicted files in tests. We test the AlreadyLocal path instead.)
func TestDownloadNonEvictedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// File is marked as evicted in the struct, but on disk it's local.
	// Download should trigger readOneByte (succeeds), then verify
	// (succeeds because it's actually local).
	f := File{Path: path, Status: StatusEvicted}
	dr := Download(f)
	if dr.Status != DownloadOK {
		t.Errorf("expected DownloadOK (file is actually local), got %d, err=%v", dr.Status, dr.Err)
	}
}

// TestVerifyPlatformOnDarwin verifies that VerifyPlatform succeeds on macOS.
func TestVerifyPlatformOnDarwin(t *testing.T) {
	err := VerifyPlatform()
	// This test only runs on macOS; on other platforms it should error.
	if err != nil && !errors.Is(err, ErrNotDarwin) {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestScanResultEvictedCount verifies the EvictedCount method.
func TestScanResultEvictedCount(t *testing.T) {
	r := ScanResult{
		Evicted: []File{{Path: "a", Status: StatusEvicted}},
	}
	if r.EvictedCount() != 1 {
		t.Errorf("expected 1, got %d", r.EvictedCount())
	}
}

// TestFileIsEvicted verifies the IsEvicted method.
func TestFileIsEvicted(t *testing.T) {
	evicted := File{Status: StatusEvicted}
	if !evicted.IsEvicted() {
		t.Error("expected true for StatusEvicted")
	}

	local := File{Status: StatusLocal}
	if local.IsEvicted() {
		t.Error("expected false for StatusLocal")
	}
}
