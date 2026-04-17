package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// osLookPath and osRunCmd are vars so tests can inject mocks.
var (
	osLookPath = exec.LookPath
	osRunCmd   = runCmdReal
)

// toolExists reports whether the named executable is on PATH.
func toolExists(name string) bool {
	_, err := osLookPath(name)
	return err == nil
}

// findPip returns the first available pip command string, or "" if none found.
// Returned value is a space-separated command string: "pip3", "pip", or "python3 -m pip".
func findPip() string {
	candidates := []struct{ bin, full string }{
		{"pip3", "pip3"},
		{"pip", "pip"},
		{"python3", "python3 -m pip"},
	}
	for _, c := range candidates {
		if _, err := osLookPath(c.bin); err == nil {
			return c.full
		}
	}
	return ""
}

// fallbackInstructions returns manual install instructions for all platforms.
func fallbackInstructions() string {
	return `Please install the missing dependencies manually:

  ffmpeg:
    macOS:   brew install ffmpeg
    Linux:   sudo apt-get install ffmpeg
    Windows: winget install --id Gyan.FFmpeg -e

  openai-whisper:
    pip3 install openai-whisper`
}

// cmdString joins name + args into a display string (used in tests and errors).
func cmdString(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

func runCmdReal(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stderr // stream to same fd as progress bar
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // needed for sudo password prompts on Linux
	return cmd.Run()
}

// installWhisperViaPip installs openai-whisper using the first available pip.
func installWhisperViaPip() error {
	pip := findPip()
	if pip == "" {
		return fmt.Errorf("no pip found\n%s", fallbackInstructions())
	}
	log.Printf("whisper not found — installing openai-whisper via %s", pip)
	parts := strings.Fields(pip)
	args := append(parts[1:], "install", "openai-whisper")
	if err := osRunCmd(parts[0], args...); err != nil {
		return fmt.Errorf("could not install openai-whisper: %w\n%s", err, fallbackInstructions())
	}
	return nil
}
