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

func installDarwin(ffmpegMissing, whisperMissing bool) error {
	if ffmpegMissing {
		if !toolExists("brew") {
			log.Printf("Homebrew not found — installing Homebrew")
			const brewScript = `curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | NONINTERACTIVE=1 /bin/bash`
			if err := osRunCmd("/bin/sh", "-c", brewScript); err != nil {
				return fmt.Errorf("could not install Homebrew: %w\nInstall manually: https://brew.sh\n%s", err, fallbackInstructions())
			}
		}
		log.Printf("ffmpeg not found — installing via Homebrew")
		if err := osRunCmd("brew", "install", "ffmpeg"); err != nil {
			return fmt.Errorf("could not install ffmpeg: %w\n%s", err, fallbackInstructions())
		}
	}
	if whisperMissing {
		return installWhisperViaPip()
	}
	return nil
}

func installLinux(ffmpegMissing, whisperMissing bool) error {
	if ffmpegMissing {
		aptCmd := "apt-get"
		if !toolExists("apt-get") {
			aptCmd = "apt"
		}
		if !toolExists(aptCmd) {
			return fmt.Errorf("could not find apt-get or apt\n%s", fallbackInstructions())
		}
		log.Printf("ffmpeg not found — installing via %s", aptCmd)
		if err := osRunCmd(aptCmd, "install", "-y", "ffmpeg"); err != nil {
			return fmt.Errorf("could not install ffmpeg: %w\n%s", err, fallbackInstructions())
		}
	}
	if whisperMissing {
		return installWhisperViaPip()
	}
	return nil
}

func installWindows(ffmpegMissing, whisperMissing bool) error {
	if ffmpegMissing {
		if !toolExists("winget") {
			return fmt.Errorf("winget not found\n%s", fallbackInstructions())
		}
		log.Printf("ffmpeg not found — installing via winget")
		if err := osRunCmd("winget", "install", "--id", "Gyan.FFmpeg", "-e"); err != nil {
			return fmt.Errorf("could not install ffmpeg: %w\n%s", err, fallbackInstructions())
		}
	}
	if whisperMissing {
		return installWhisperViaPip()
	}
	return nil
}
