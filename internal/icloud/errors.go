// errors.go defines sentinel errors for package icloud.
// It is part of package icloud; see icloud.go for the package overview.

package icloud

// Errors specific to scanning.
// This file is intentionally separate from icloud.go to keep the
// sentinel-error list and the scan logic in their own SRP units.

import "errors"

var (
	// ErrScanPath is returned when the scan root cannot be resolved
	// to an absolute path or does not exist.
	ErrScanPath = errors.New("scan path is invalid or inaccessible")

	// ErrStillDataless is returned when a file remains evicted after
	// the download trigger. This typically indicates a network timeout
	// or an iCloud sync error.
	ErrStillDataless = errors.New("file is still dataless after download trigger")
)
