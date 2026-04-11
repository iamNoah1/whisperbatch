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

// RunBatch transcribes all files using a fixed-size worker pool.
// Progress is displayed on stderr via a progress bar. Failed files are
// printed to stderr as they occur without interrupting the bar.
func RunBatch(files []string, cfg Config) Summary {
	jobs := make(chan string, len(files))
	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	var (
		mu      sync.Mutex
		results = make([]Result, 0, len(files))
		wg      sync.WaitGroup
	)

	bar := progressbar.NewOptions(
		len(files),
		progressbar.OptionSetDescription("Transcribing"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
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
				r := processFile(inputPath, cfg)
				_ = bar.Add(1)
				if r.Err != nil {
					fmt.Fprintf(os.Stderr, "\nFAILED %s: %v\n", filepath.Base(inputPath), r.Err)
				}
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
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
