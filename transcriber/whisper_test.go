package transcriber

import (
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestBuildArgs verifies the exact flags passed to the whisper subprocess.
// This is the primary guard against invalid flag regressions (e.g. --language auto,
// --device auto) that only surface at runtime when whisper is actually called.
func TestBuildArgs(t *testing.T) {
	args := buildArgs("/audio/test.mp3", "/out", "tiny", []string{"txt", "srt"})

	// First arg must be the input file.
	if args[0] != "/audio/test.mp3" {
		t.Errorf("args[0] = %q, want input path", args[0])
	}

	must := map[string]string{
		"--model":      "tiny",
		"--output_dir": "/out",
	}
	for i := 0; i < len(args)-1; i++ {
		if want, ok := must[args[i]]; ok {
			if args[i+1] != want {
				t.Errorf("flag %s = %q, want %q", args[i], args[i+1], want)
			}
			delete(must, args[i])
		}
	}
	for flag := range must {
		t.Errorf("required flag %s not found in args", flag)
	}

	// Verify output formats are present.
	if !slices.Contains(args, "txt") || !slices.Contains(args, "srt") {
		t.Errorf("expected output formats in args, got %v", args)
	}

	// Guard against invalid flag values that break whisper/PyTorch.
	banned := []string{"--language", "--device"}
	for _, flag := range banned {
		if slices.Contains(args, flag) {
			t.Errorf("args must not contain %q (causes whisper/PyTorch errors)", flag)
		}
	}
}

// TestTranscribeIntegration runs whisper-ctranslate2 on a minimal WAV file.
// It is skipped when whisper-ctranslate2 is not installed, so CI without it still
// passes. Run locally to catch flag or invocation regressions before cutting a
// release.
func TestTranscribeIntegration(t *testing.T) {
	if _, err := exec.LookPath("whisper-ctranslate2"); err != nil {
		t.Skip("whisper-ctranslate2 not in PATH — skipping integration test")
	}

	dir := t.TempDir()
	wavPath := filepath.Join(dir, "silence.wav")
	if err := os.WriteFile(wavPath, minimalWAV(), 0o644); err != nil {
		t.Fatalf("writing test WAV: %v", err)
	}

	if err := Transcribe(wavPath, dir, "tiny", []string{"txt"}, time.Minute); err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	outPath := filepath.Join(dir, "silence.txt")
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Errorf("expected output file %s was not created", outPath)
	}
}

func TestTranscribeError_withStderr(t *testing.T) {
	e := &TranscribeError{File: "/audio/test.mp3", Stderr: "out of memory"}
	got := e.Error()
	if !strings.Contains(got, "test.mp3") || !strings.Contains(got, "out of memory") {
		t.Errorf("unexpected error string: %q", got)
	}
}

func TestTranscribeError_withoutStderr(t *testing.T) {
	e := &TranscribeError{File: "/audio/test.mp3", Stderr: ""}
	got := e.Error()
	if !strings.Contains(got, "test.mp3") {
		t.Errorf("unexpected error string: %q", got)
	}
	if strings.Contains(got, ":") && strings.HasSuffix(got, ":") {
		t.Errorf("trailing colon in error string without stderr: %q", got)
	}
}

func TestTranscribeError_Unwrap(t *testing.T) {
	cause := errors.New("exit status 1")
	e := &TranscribeError{Cause: cause}
	if e.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", e.Unwrap(), cause)
	}
}

func TestTranscribeError_timedOut(t *testing.T) {
	e := &TranscribeError{
		File:     "/audio/long.mp3",
		TimedOut: true,
		Timeout:  4 * time.Hour,
		// Stderr deliberately set to a benign warning to prove the timeout
		// message replaces it instead of being masked by it.
		Stderr: "Warning: unauthenticated HF request",
	}
	got := e.Error()
	if !strings.Contains(got, "timed out") {
		t.Errorf("expected timeout message, got: %q", got)
	}
	if !strings.Contains(got, "4h0m0s") {
		t.Errorf("expected timeout duration in message, got: %q", got)
	}
	if !strings.Contains(got, "long.mp3") {
		t.Errorf("expected file name in message, got: %q", got)
	}
	if strings.Contains(got, "unauthenticated") {
		t.Errorf("timeout message must not leak stderr warnings, got: %q", got)
	}
}

// minimalWAV returns a valid 44-byte WAV header with ~0.1 s of silence (PCM
// 16-bit mono 8 kHz). Small enough for whisper to process in seconds.
func minimalWAV() []byte {
	const sampleRate = 8000
	const numSamples = sampleRate / 10 // 0.1 s
	const dataSize = numSamples * 2    // 16-bit samples

	buf := make([]byte, 44+dataSize)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16) // PCM chunk size
	binary.LittleEndian.PutUint16(buf[20:22], 1)  // PCM format
	binary.LittleEndian.PutUint16(buf[22:24], 1)  // mono
	binary.LittleEndian.PutUint32(buf[24:28], sampleRate)
	binary.LittleEndian.PutUint32(buf[28:32], sampleRate*2) // byte rate
	binary.LittleEndian.PutUint16(buf[32:34], 2)            // block align
	binary.LittleEndian.PutUint16(buf[34:36], 16)           // bits per sample
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	// remaining bytes are zero — silence
	return buf
}
