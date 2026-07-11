// Package cli implements the command-line interface for icloud-pull.
//
// This package is the presentation layer: it parses os.Args, delegates
// to the icloud package for all actual work, and prints results via
// the msg package. It contains no business logic.
//
// Subcommand structure (following OCP - new commands are added as
// new cases, not by modifying existing ones):
//
//	icloud-pull status  <path> [--json]       Scan and report evicted files
//	icloud-pull download <path> [--concurrency N] [--json]  Download evicted files
//	icloud-pull --version                       Print version
//	icloud-pull --help                          Show help
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hyper0x/icloud-pull/internal/icloud"
	"github.com/hyper0x/icloud-pull/internal/msg"
)

// Version and Commit are set via -ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
)

// Run is the main entry point. It parses args, dispatches to the
// appropriate handler, and returns an exit code.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Print(msg.Usage)
		return 1
	}

	switch args[0] {
	case "--help", "-h", "help":
		fmt.Print(msg.Banner)
		fmt.Println()
		fmt.Print(msg.Usage)
		return 0
	case "--version", "-v", "version":
		fmt.Printf("icloud-pull %s (commit: %s)\n", Version, Commit)
		return 0
	case "status":
		return runStatus(args[1:])
	case "download":
		return runDownload(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command '%s'\n", args[0])
		fmt.Fprint(os.Stderr, msg.Usage)
		return 1
	}
}

// runStatus scans a directory and reports evicted files.
func runStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: icloud-pull status <path> [--json]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Scan a directory and report evicted files.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  icloud-pull status ~/Documents")
	}
	jsonOut := fs.Bool("json", false, "output results as JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	roots := fs.Args()
	if len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "error: no path specified")
		fs.Usage()
		return 1
	}

	if err := icloud.VerifyPlatform(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	exitCode := 0
	for _, root := range roots {
		result, err := icloud.Scan(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			exitCode = 1
			continue
		}

		if *jsonOut {
			printStatusJSON(result)
		} else {
			printStatusText(result)
		}
	}

	return exitCode
}

// runDownload scans and downloads all evicted files.
func runDownload(args []string) int {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: icloud-pull download <path> [--concurrency N] [--json]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Scan and download all evicted files under <path>.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  icloud-pull download --concurrency 10 ~/Documents")
	}
	concurrency := fs.Int("concurrency", 5, "maximum simultaneous downloads")
	jsonOut := fs.Bool("json", false, "output results as JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	roots := fs.Args()
	if len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "error: no path specified")
		fs.Usage()
		return 1
	}

	if err := icloud.VerifyPlatform(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	exitCode := 0
	for _, root := range roots {
		if rc := processDownload(root, *concurrency, *jsonOut); rc != 0 {
			exitCode = rc
		}
	}

	return exitCode
}

// processDownload scans a root, downloads evicted files, and reports.
func processDownload(root string, concurrency int, jsonOut bool) int {
	result, err := icloud.Scan(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if result.EvictedCount() == 0 {
		if jsonOut {
			printDownloadJSON(result.Root, nil)
		} else {
			fmt.Printf("Scanned %d files in %s, nothing evicted.\n", result.Total, root)
		}
		return 0
	}

	if !jsonOut {
		fmt.Printf("Scanning: %s\n", root)
		fmt.Print(msg.ScanReport(result.Total, result.EvictedCount()))
		fmt.Printf("Downloading %d files (concurrency: %d)...\n", result.EvictedCount(), concurrency)
	}

	// Concurrent download with progress callback.
	// Note: the callback only prints; it does NOT accumulate counters
	// because DownloadAll's goroutines would race on shared state.
	// SummarizeResults (called after all goroutines finish) is the
	// authoritative source for the final tally.
	results := icloud.DownloadAll(result.Evicted, concurrency, func(_, _ int, dr icloud.DownloadResult) {
		if jsonOut {
			return // suppress text progress in JSON mode
		}
		switch dr.Status {
		case icloud.DownloadOK:
			fmt.Print(msg.DownloadOK(dr.File.Path))
		case icloud.DownloadFailed:
			fmt.Print(msg.DownloadFail(dr.File.Path, dr.Err))
		case icloud.DownloadAlreadyLocal:
			fmt.Print(msg.DownloadSkip(dr.File.Path))
		}
	})

	if jsonOut {
		printDownloadJSON(result.Root, results)
	} else {
		summary := icloud.SummarizeResults(results)
		fmt.Print(msg.Done(summary.Downloaded, summary.Failed, summary.Skipped))
	}

	if hasFailures(results) {
		return 1
	}
	return 0
}

// hasFailures checks whether any result failed.
func hasFailures(results []icloud.DownloadResult) bool {
	for _, dr := range results {
		if dr.Status == icloud.DownloadFailed {
			return true
		}
	}
	return false
}

// --- JSON output ---

// statusJSON is the JSON representation of a scan result.
type statusJSON struct {
	Root    string `json:"root"`
	Total   int    `json:"total"`
	Evicted int    `json:"evicted"`
	Files   []fileJSON `json:"files"`
}

// fileJSON is the JSON representation of an evicted file.
type fileJSON struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func printStatusJSON(r *icloud.ScanResult) {
	out := statusJSON{
		Root:    r.Root,
		Total:   r.Total,
		Evicted: r.EvictedCount(),
		Files:   make([]fileJSON, 0, len(r.Evicted)),
	}
	for _, f := range r.Evicted {
		out.Files = append(out.Files, fileJSON{Path: f.Path, Size: f.Size})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

// downloadJSON is the JSON representation of a download result set.
type downloadJSON struct {
	Root    string               `json:"root"`
	Summary downloadSummaryJSON  `json:"summary"`
	Results []downloadItemJSON   `json:"results"`
}

type downloadSummaryJSON struct {
	Downloaded int `json:"downloaded"`
	Failed     int `json:"failed"`
	Skipped    int `json:"skipped"`
}

type downloadItemJSON struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func printDownloadJSON(root string, results []icloud.DownloadResult) {
	summary := icloud.SummarizeResults(results)
	out := downloadJSON{
		Root: root,
		Summary: downloadSummaryJSON{
			Downloaded: summary.Downloaded,
			Failed:     summary.Failed,
			Skipped:    summary.Skipped,
		},
		Results: make([]downloadItemJSON, 0, len(results)),
	}

	for _, dr := range results {
		item := downloadItemJSON{
			Path:   dr.File.Path,
			Status: dr.Status.String(),
		}
		if dr.Err != nil {
			item.Error = dr.Err.Error()
		}
		out.Results = append(out.Results, item)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

// --- Text output helpers ---

func printStatusText(r *icloud.ScanResult) {
	fmt.Print(msg.ScanReport(r.Total, r.EvictedCount()))
	if r.EvictedCount() > 0 {
		fmt.Println("Evicted files:")
		for _, f := range r.Evicted {
			fmt.Printf("  %s (%d bytes)\n", f.Path, f.Size)
		}
	}
}
