// scan.go implements the directory walker that detects evicted files.
// It is part of package icloud; see icloud.go for the package overview.

package icloud

// scan.go implements the directory walker that detects evicted files.
//
// Design decisions:
//   - Detection uses syscall.Stat_t.Flags (BSD file flags), not stat -L.
//     This avoids spawning a subprocess per file.
//   - We use filepath.WalkDir (not filepath.Walk) because WalkDir is
//     more efficient: it doesn't call os.Lstat on every file (the
//     WalkDirFunc already receives a DirEntry backed by the same stat).
//   - Symlinks are skipped (os.Lstat sees the link, not the target;
//     pulling a symlink makes no sense in iCloud context).

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// ScanResult holds the outcome of a directory scan.
type ScanResult struct {
	Root    string
	Total   int // total files scanned (excluding symlinks and directories)
	Evicted []File
	Local   []File
}

// EvictedCount returns the number of evicted files found.
func (r ScanResult) EvictedCount() int {
	return len(r.Evicted)
}

// Scan walks the directory tree rooted at root and classifies every
// regular file as either local or evicted.
//
// Symlinks are skipped: iCloud eviction applies to regular files, and
// following symlinks could pull unintended data or loop.
func Scan(root string) (*ScanResult, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScanPath, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScanPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: not a directory", ErrScanPath)
	}

	result := &ScanResult{Root: abs}

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// WalkDir passes errors from the OS; we report but continue.
			return err
		}

		// Skip symlinks - they are not iCloud-evictable regular files.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		// WalkDir calls the callback for directories too; skip them.
		if d.IsDir() {
			return nil
		}

		// We need the full stat (not just DirEntry) to read BSD flags.
		fi, err := os.Lstat(path)
		if err != nil {
			// File may have been deleted between WalkDir and Lstat.
			// Skip it rather than aborting the entire scan.
			//nolint:nilerr // intentional: skip missing files, don't fail the scan
			return nil
		}

		result.Total++

		f := File{
			Path: path,
			Size: fi.Size(),
		}

		if isDataless(fi) {
			f.Status = StatusEvicted
			result.Evicted = append(result.Evicted, f)
		} else {
			f.Status = StatusLocal
			result.Local = append(result.Local, f)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	return result, nil
}

// isDataless checks whether a file's BSD flags include UF_DATALESS.
//
// On macOS, syscall.Stat_t contains a Flags field that holds the
// BSD file flags (UF_* and SF_*). The UF_DATALESS flag (0x40000000)
// is set by the kernel when file content is offloaded to iCloud.
//
// On non-Darwin platforms this function always returns false, but
// VerifyPlatform() should have already rejected non-macOS usage.
func isDataless(fi os.FileInfo) bool {
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return stat.Flags&UFDataless != 0
}
