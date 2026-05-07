package transcriber

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// DefaultTimeout is the per-file timeout used when none is configured.
const DefaultTimeout = 4 * time.Hour

// TranscribeError holds structured details about a failed whisper invocation.
type TranscribeError struct {
	File     string
	Stderr   string
	Cause    error
	TimedOut bool
	Timeout  time.Duration
}

func (e *TranscribeError) Error() string {
	if e.TimedOut {
		return fmt.Sprintf(
			"whisper timed out after %s for %q — increase --timeout for longer audio",
			e.Timeout.Round(time.Second), e.File,
		)
	}
	msg := fmt.Sprintf("whisper failed for %q", e.File)
	if e.Stderr != "" {
		msg += ": " + e.Stderr
	}
	return msg
}

func (e *TranscribeError) Unwrap() error { return e.Cause }

// buildArgs constructs the argument slice for the whisper-ctranslate2 invocation.
func buildArgs(inputPath, outputDir, model string, formats []string) []string {
	args := []string{
		inputPath,
		"--model", model,
		"--output_dir", outputDir,
	}
	for _, f := range formats {
		args = append(args, "--output_format", f)
	}
	return args
}

// Transcribe runs the whisper-ctranslate2 (faster-whisper) CLI on a single audio
// file, writing outputs to outputDir in each of the requested formats.
//
// timeout is the maximum wall-clock duration for the subprocess. When it
// expires the process is killed and a TranscribeError with TimedOut=true is
// returned so callers can distinguish a hang/long file from a real failure.
// A non-positive timeout falls back to DefaultTimeout.
func Transcribe(inputPath, outputDir, model string, formats []string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := buildArgs(inputPath, outputDir, model, formats)

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "whisper-ctranslate2", args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return &TranscribeError{
				File:     inputPath,
				Cause:    ctx.Err(),
				TimedOut: true,
				Timeout:  timeout,
			}
		}
		stderrStr := bytes.TrimSpace(stderr.Bytes())
		return &TranscribeError{
			File:   inputPath,
			Stderr: string(stderrStr),
			Cause:  err,
		}
	}
	return nil
}
