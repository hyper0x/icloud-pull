package icloud

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestDownloadAllSequential tests DownloadAll with concurrency=1.
func TestDownloadAllSequential(t *testing.T) {
	dir := t.TempDir()
	var files []File
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, File{Path: path, Status: StatusEvicted})
	}

	results := DownloadAll(files, 1, nil)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// All files are actually local on disk, so they should download OK.
	for i, dr := range results {
		if dr.Status != DownloadOK {
			t.Errorf("result[%d]: expected DownloadOK, got %s, err=%v", i, dr.Status, dr.Err)
		}
	}

	// Results should be in the same order as input.
	for i, f := range files {
		if results[i].File.Path != f.Path {
			t.Errorf("result[%d]: expected path %s, got %s", i, f.Path, results[i].File.Path)
		}
	}
}

// TestDownloadAllConcurrent tests DownloadAll with concurrency > 1.
func TestDownloadAllConcurrent(t *testing.T) {
	dir := t.TempDir()
	var files []File
	for i := range 20 {
		path := filepath.Join(dir, "file"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, File{Path: path, Status: StatusEvicted})
	}

	results := DownloadAll(files, 5, nil)
	if len(results) != 20 {
		t.Fatalf("expected 20 results, got %d", len(results))
	}

	for i, dr := range results {
		if dr.Status != DownloadOK {
			t.Errorf("result[%d]: expected DownloadOK, got %s", i, dr.Status)
		}
	}
}

// TestDownloadAllProgressCallback verifies the progress callback is called.
func TestDownloadAllProgressCallback(t *testing.T) {
	dir := t.TempDir()
	var files []File
	for range 5 {
		path := filepath.Join(dir, "file.txt")
		_ = os.WriteFile(path, []byte("x"), 0o644) // same file, that's fine
		files = append(files, File{Path: path, Status: StatusEvicted})
	}

	var mu sync.Mutex
	callCount := 0
	DownloadAll(files, 3, func(idx, total int, dr DownloadResult) {
		mu.Lock()
		callCount++
		mu.Unlock()
		if total != 5 {
			t.Errorf("expected total=5, got %d", total)
		}
	})

	mu.Lock()
	if callCount != 5 {
		t.Errorf("expected 5 progress calls, got %d", callCount)
	}
	mu.Unlock()
}

// TestDownloadAllEmpty tests DownloadAll with an empty file list.
func TestDownloadAllEmpty(t *testing.T) {
	results := DownloadAll(nil, 5, nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(results))
	}
}

// TestDownloadAllZeroConcurrency tests that concurrency <= 0 defaults to 1.
func TestDownloadAllZeroConcurrency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []File{{Path: path, Status: StatusEvicted}}

	results := DownloadAll(files, 0, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != DownloadOK {
		t.Errorf("expected DownloadOK, got %s", results[0].Status)
	}
}

// TestSummarizeResults verifies the summary aggregation.
func TestSummarizeResults(t *testing.T) {
	results := []DownloadResult{
		{Status: DownloadOK},
		{Status: DownloadOK},
		{Status: DownloadFailed, Err: ErrStillDataless},
		{Status: DownloadAlreadyLocal},
	}

	s := SummarizeResults(results)
	if s.Downloaded != 2 {
		t.Errorf("expected 2 downloaded, got %d", s.Downloaded)
	}
	if s.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", s.Failed)
	}
	if s.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", s.Skipped)
	}
}

// TestSummarizeResultsEmpty verifies empty input.
func TestSummarizeResultsEmpty(t *testing.T) {
	s := SummarizeResults(nil)
	if s.Downloaded != 0 || s.Failed != 0 || s.Skipped != 0 {
		t.Errorf("expected all zeros, got %+v", s)
	}
}

// TestDownloadStatusString verifies the String() method for all statuses.
func TestDownloadStatusString(t *testing.T) {
	tests := []struct {
		status DownloadStatus
		want   string
	}{
		{DownloadOK, "ok"},
		{DownloadAlreadyLocal, "already_local"},
		{DownloadFailed, "failed"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

// TestDownloadNonExistentFile verifies that Download on a nonexistent file
// returns DownloadFailed.
func TestDownloadNonExistentFile(t *testing.T) {
	f := File{Path: "/nonexistent/file/abc123.txt", Status: StatusEvicted}
	dr := Download(f)
	if dr.Status != DownloadFailed {
		t.Errorf("expected DownloadFailed, got %s", dr.Status)
	}
	if dr.Err == nil {
		t.Error("expected non-nil error for nonexistent file")
	}
}
