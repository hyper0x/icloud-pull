// Package msg holds all user-visible strings for icloud-pull.
//
// Centralizing messages here follows the Information Expert principle:
// the message text lives in one place, making it easy to review and
// update wording without touching logic. This also makes future
// localization straightforward.
package msg

import "fmt"

// Banner is printed at program start.
const Banner = `icloud-pull - download evicted iCloud files`

// Usage explains how to invoke the tool.
const Usage = `Usage:
  icloud-pull status <path> [--json]
          Scan a directory and report evicted files.

  icloud-pull download <path> [--concurrency N] [--json]
          Scan and download all evicted files under <path>.

  icloud-pull --version
          Print version information.

  icloud-pull --help
          Show this help message.

Examples:
  icloud-pull status ~/Documents
  icloud-pull status --json ~/Library/Mobile\ Documents
  icloud-pull download --concurrency 10 ~/Documents
  icloud-pull download --json ~/Library/Mobile\ Documents
`

// ScanReport prints a summary of the scan result.
func ScanReport(total, evicted int) string {
	return fmt.Sprintf("Scanned %d files, found %d evicted.\n", total, evicted)
}

// DownloadOK reports a successful download.
func DownloadOK(path string) string {
	return fmt.Sprintf("  ✓ %s\n", path)
}

// DownloadFail reports a failed download.
func DownloadFail(path string, err error) string {
	return fmt.Sprintf("  ✗ %s: %v\n", path, err)
}

// DownloadSkip reports a file that was already local.
func DownloadSkip(path string) string {
	return fmt.Sprintf("  - %s (already local)\n", path)
}

// Done is printed when all downloads complete.
func Done(ok, fail, skip int) string {
	return fmt.Sprintf("\nDone: %d downloaded, %d failed, %d skipped.\n", ok, fail, skip)
}
