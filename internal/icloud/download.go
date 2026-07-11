// download.go implements the file download (de-eviction) logic.
// It is part of package icloud; see icloud.go for the package overview.

package icloud

// download.go implements the file download (de-eviction) logic.
//
// Mechanism: on macOS, reading even a single byte from an evicted
// (dataless) file triggers APFS to transparently fetch the content
// from iCloud. The read call blocks until the download completes.
//
// Design:
//   - ReadOneByte is the low-level primitive: it opens the file and
//     reads exactly one byte. This is the APFS fetch trigger.
//   - Download is the public API: it reads one byte, then verifies
//     the file is no longer dataless.
//   - DownloadResult captures the outcome for reporting.
//
// Why not os.ReadFile? os.ReadFile allocates a buffer matching the
// file's reported size. For an evicted file, stat reports the original
// (potentially large) size, but the read blocks until the content
// arrives. Using ReadOneByte avoids allocating a large buffer; the
// single-byte read is sufficient to trigger the fetch, and macOS
// handles the rest internally.

import (
	"fmt"
	"os"
)

// DownloadStatus represents the outcome of a download attempt.
type DownloadStatus int

const (
	// DownloadOK means the file was successfully de-evicted.
	DownloadOK DownloadStatus = iota
	// DownloadAlreadyLocal means the file was not evicted; no action taken.
	DownloadAlreadyLocal
	// DownloadFailed means the read triggered but verification still
	// shows dataless (network timeout, iCloud error, etc.).
	DownloadFailed
)

// DownloadResult captures the outcome of downloading a single file.
type DownloadResult struct {
	File   File
	Status DownloadStatus
	Err    error // populated when Status == DownloadFailed
}

// Download triggers APFS to fetch the content of an evicted file.
//
// It reads one byte from the file (which blocks until the content
// arrives from iCloud), then re-stats the file to confirm the
// dataless flag is cleared.
//
// If the file is already local, it returns DownloadAlreadyLocal
// without touching the file.
func Download(f File) DownloadResult {
	if !f.IsEvicted() {
		return DownloadResult{File: f, Status: DownloadAlreadyLocal}
	}

	if err := readOneByte(f.Path); err != nil {
		return DownloadResult{
			File:   f,
			Status: DownloadFailed,
			Err:    fmt.Errorf("read trigger: %w", err),
		}
	}

	// Verify the file is no longer dataless.
	fi, err := os.Lstat(f.Path)
	if err != nil {
		return DownloadResult{
			File:   f,
			Status: DownloadFailed,
			Err:    fmt.Errorf("post-download stat: %w", err),
		}
	}

	if isDataless(fi) {
		return DownloadResult{
			File:   f,
			Status: DownloadFailed,
			Err:    ErrStillDataless,
		}
	}

	return DownloadResult{File: f, Status: DownloadOK}
}

// readOneByte opens the file and reads exactly one byte.
// The read blocks until APFS fetches the content from iCloud.
// On success, the file content is now fully present on disk (not just
// the single byte - macOS fetches the entire file).
func readOneByte(path string) error {
	file, err := os.Open(path) //nolint:gosec // path comes from our own directory scan, not user input
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	buf := make([]byte, 1)
	n, err := file.Read(buf)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("unexpected read count: %d (expected 1)", n)
	}

	return nil
}
