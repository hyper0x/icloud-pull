// Package icloud provides utilities to detect and download evicted
// (dataless) files on macOS iCloud Drive.
//
// macOS storage optimization can offload file contents to iCloud while
// keeping only metadata locally. Such files carry the UF_DATALESS BSD
// flag (0x40000000). They appear in ls and stat with their original
// size, but reading them returns zero bytes.
//
// The core mechanism for downloading: reading a single byte from an
// evicted file triggers APFS transparent fetch. macOS then downloads
// the file content from iCloud in the background.
package icloud

import (
	"errors"
	"runtime"
)

// UFDataless is the BSD file flag marking an evicted (dataless) file.
// On macOS, the system sets this flag when it offloads file contents
// to iCloud, keeping only metadata locally.
//
// Defined in <sys/stat.h> as UF_DATALESS.
const UFDataless = 0x40000000

// Sentinel errors. Using errors.New (not fmt.Errorf) because these are
// static strings with no formatting.
var (
	// ErrNotDarwin is returned when the tool is run on a non-macOS system.
	ErrNotDarwin = errors.New("icloud-pull only works on macOS (APFS eviction is a Darwin feature)")
)

// FileStatus represents the eviction state of a single file.
type FileStatus int

const (
	// StatusLocal means the file content is fully present on disk.
	StatusLocal FileStatus = iota
	// StatusEvicted means the file content has been offloaded to iCloud.
	// The file metadata exists locally, but reading returns zero bytes.
	StatusEvicted
)

// File represents a scanned file with its eviction status.
type File struct {
	Path   string
	Status FileStatus
	Size   int64 // original file size from stat (may be non-zero even when evicted)
}

// IsEvicted returns true if the file's content has been offloaded.
func (f File) IsEvicted() bool {
	return f.Status == StatusEvicted
}

// String returns a human-readable status name for JSON output.
func (s DownloadStatus) String() string {
	switch s {
	case DownloadOK:
		return "ok"
	case DownloadAlreadyLocal:
		return "already_local"
	case DownloadFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// VerifyPlatform returns an error if the current OS is not macOS.
// iCloud eviction is an APFS feature; the tool cannot function elsewhere.
func VerifyPlatform() error {
	if runtime.GOOS != "darwin" {
		return ErrNotDarwin
	}
	return nil
}
