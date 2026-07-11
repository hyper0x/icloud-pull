// batch.go implements concurrent batch download of evicted files.
// It is part of package icloud; see icloud.go for the package overview.

package icloud

// Design:
//   - DownloadAll fans out Download calls across a goroutine pool
//     bounded by the concurrency parameter.
//   - Results are collected in order via an indexed channel, so the
//     output matches the input order regardless of completion order.
//   - The progress callback is invoked after each file completes,
//     enabling the CLI layer to print progress without coupling to
//     the download internals (Tell Don't Ask).

import "sync"

// ProgressFn is called after each file download attempt.
// index is the 0-based position in the files slice.
// total is the total number of files being processed.
type ProgressFn func(index, total int, result DownloadResult)

// DownloadAll downloads all evicted files concurrently.
//
// concurrency controls the maximum number of simultaneous downloads.
// A value <= 0 defaults to 1 (sequential).
//
// progress is called after each file completes (may be nil).
//
// Returns results in the same order as the input files.
func DownloadAll(files []File, concurrency int, progress ProgressFn) []DownloadResult {
	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > len(files) {
		concurrency = len(files)
	}

	results := make([]DownloadResult, len(files))
	total := len(files)

	type slot struct {
		index  int
		result DownloadResult
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	ch := make(chan slot, total)

	for i, f := range files {
		wg.Add(1)
		go func(idx int, file File) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			dr := Download(file)
			ch <- slot{index: idx, result: dr}

			if progress != nil {
				progress(idx, total, dr)
			}
		}(i, f)
	}

	// Close channel after all goroutines finish.
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results.
	for s := range ch {
		results[s.index] = s.result
	}

	return results
}

// Summary aggregates download results for reporting.
type Summary struct {
	Downloaded int
	Failed     int
	Skipped    int
}

// SummarizeResults tallies a slice of DownloadResult.
func SummarizeResults(results []DownloadResult) Summary {
	var s Summary
	for _, dr := range results {
		switch dr.Status {
		case DownloadOK:
			s.Downloaded++
		case DownloadFailed:
			s.Failed++
		case DownloadAlreadyLocal:
			s.Skipped++
		}
	}
	return s
}
