package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const whisperTimeout = 30 * time.Minute

// TranscribeError holds structured details about a failed whisper invocation.
type TranscribeError struct {
	File   string
	Stderr string
	Cause  error
}

func (e *TranscribeError) Error() string {
	msg := fmt.Sprintf("whisper failed for %q", e.File)
	if e.Stderr != "" {
		msg += ": " + e.Stderr
	}
	return msg
}

func (e *TranscribeError) Unwrap() error { return e.Cause }

// Transcribe runs the whisper CLI on a single audio file, writing outputs to
// outputDir in each of the requested formats.
//
// It uses a 30-minute per-file timeout and captures stderr for error reporting.
// Multiple --output_format flags are passed for tools that support it (e.g.
// faster-whisper). For openai-whisper (which accepts only one format per run),
// pass a single format or use --format all and prune outputs afterward.
func Transcribe(inputPath, outputDir, model string, formats []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), whisperTimeout)
	defer cancel()

	args := []string{
		inputPath,
		"--model", model,
		"--output_dir", outputDir,
		"--language", "auto",
		"--device", "auto",
	}
	for _, f := range formats {
		args = append(args, "--output_format", f)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "whisper", args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := bytes.TrimSpace(stderr.Bytes())
		return &TranscribeError{
			File:   inputPath,
			Stderr: string(stderrStr),
			Cause:  err,
		}
	}
	return nil
}
