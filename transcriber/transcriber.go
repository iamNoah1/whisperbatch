package transcriber

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/iamNoah1/whisperbatch/fileutil"
	"github.com/schollz/progressbar/v3"
)

// Config holds the parameters for a batch transcription run.
type Config struct {
	OutputDir string
	Formats   []string
	Workers   int
	Model     string
	Overwrite bool
}

// Result captures the outcome of transcribing a single file.
type Result struct {
	File    string
	Elapsed time.Duration
	Err     error
}

// Summary holds aggregate statistics for the completed batch.
type Summary struct {
	Total     int
	Succeeded int
	Failed    int
	TotalWall time.Duration
	Results   []Result
}

// truncateName shortens s to at most n runes, appending "…" when cut.
func truncateName(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// RunBatch transcribes all files using a fixed-size worker pool.
// Progress is displayed on stderr via a progress bar. Failed files are
// printed to stderr as they occur without interrupting the bar.
func RunBatch(files []string, cfg Config) Summary {
	jobs := make(chan string, len(files))
	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	total := len(files)
	// Each file owns 100 bar units so intra-file progress is visible.
	barMax := total * 100

	var (
		mu               sync.Mutex
		results          = make([]Result, 0, total)
		completedElapsed []time.Duration // elapsed per completed file
		wg               sync.WaitGroup
	)

	// barStats returns the number of completed files, the average elapsed
	// time per completed file (0 if none), and an ETA string.
	barStats := func() (completed int, avg time.Duration, eta string) {
		mu.Lock()
		n := len(completedElapsed)
		var sum time.Duration
		for _, d := range completedElapsed {
			sum += d
		}
		mu.Unlock()
		if n == 0 {
			return 0, 0, ""
		}
		avg = sum / time.Duration(n)
		remaining := total - n
		if remaining > 0 {
			est := avg * time.Duration(remaining)
			eta = fmt.Sprintf("~%s", est.Round(time.Second))
		}
		return n, avg, eta
	}

	// fileFraction estimates how far through a single file we are (0–1),
	// capped at 0.95 so the bar never falsely reaches the next boundary.
	// When no history is available a hyperbolic curve is used so the bar
	// still moves from the very first second.
	fileFraction := func(elapsed time.Duration, avg time.Duration) float64 {
		if avg > 0 {
			f := elapsed.Seconds() / avg.Seconds()
			if f > 0.95 {
				return 0.95
			}
			return f
		}
		// No history yet: t/(t+k) gives a curve that starts fast then
		// slows, approaching 1 asymptotically. k=90s feels natural for
		// typical Whisper jobs.
		k := 90.0
		return elapsed.Seconds() / (elapsed.Seconds() + k)
	}

	bar := progressbar.NewOptions(
		barMax,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for inputPath := range jobs {
				name := truncateName(filepath.Base(inputPath), 24)
				fileStart := time.Now()

				// barBase is the bar position at the moment this file starts.
				// The ticker only ever sets values in [barBase, barBase+95] so
				// it can never reach into the next file's territory, even if
				// completedElapsed is updated before the goroutine exits.
				mu.Lock()
				barBase := len(completedElapsed) * 100
				mu.Unlock()

				tickDone := make(chan struct{})
				go func() {
					const spinFrames = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
					frames := []rune(spinFrames)
					var frame int
					ticker := time.NewTicker(100 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-tickDone:
							return
						case <-ticker.C:
							elapsed := time.Since(fileStart)
							_, avg, eta := barStats()
							frac := fileFraction(elapsed, avg)
							target := barBase + int(frac*100)
							_ = bar.Set(target)

							spin := string(frames[frame%len(frames)])
							frame++
							elapsedRounded := elapsed.Round(time.Second)
							fileNum := barBase/100 + 1
							var desc string
							if eta != "" {
								desc = fmt.Sprintf("%s [%d/%d] %-24s %4s  %s left", spin, fileNum, total, name, elapsedRounded, eta)
							} else {
								desc = fmt.Sprintf("%s [%d/%d] %-24s %4s", spin, fileNum, total, name, elapsedRounded)
							}
							bar.Describe(desc)
						}
					}
				}()

				r := processFile(inputPath, cfg)
				close(tickDone)

				mu.Lock()
				completedElapsed = append(completedElapsed, r.Elapsed)
				results = append(results, r)
				n := len(completedElapsed)
				mu.Unlock()

				// Snap to the exact completion boundary for this file.
				_ = bar.Set(n * 100)
				if r.Err != nil {
					fmt.Fprintf(os.Stderr, "\nFAILED %s: %v\n", name, r.Err)
				}
			}
		}()
	}

	wg.Wait()
	_ = bar.Finish()

	summary := Summary{
		Total:   len(results),
		Results: results,
	}
	for _, r := range results {
		if r.Err == nil {
			summary.Succeeded++
		} else {
			summary.Failed++
		}
	}
	return summary
}

// processFile handles a single file: checks for existing outputs, runs Whisper,
// and returns a Result.
func processFile(inputPath string, cfg Config) Result {
	start := time.Now()

	if !cfg.Overwrite && allOutputsExist(inputPath, cfg.OutputDir, cfg.Formats) {
		fmt.Fprintf(
			os.Stderr,
			"\nSKIP %s: all output files already exist (use --overwrite to replace)\n",
			filepath.Base(inputPath),
		)
		return Result{File: inputPath, Elapsed: time.Since(start)}
	}

	err := Transcribe(inputPath, cfg.OutputDir, cfg.Model, cfg.Formats)
	return Result{
		File:    inputPath,
		Elapsed: time.Since(start),
		Err:     err,
	}
}

// allOutputsExist returns true when every requested format already has an
// output file on disk.
func allOutputsExist(inputPath, outputDir string, formats []string) bool {
	for _, format := range formats {
		outPath := fileutil.OutputPath(inputPath, outputDir, format)
		if _, err := os.Stat(outPath); os.IsNotExist(err) {
			return false
		}
	}
	return len(formats) > 0
}
