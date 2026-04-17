# Auto-install Dependencies Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Auto-install `ffmpeg` and `openai-whisper` on first run so users never need to set up dependencies manually.

**Architecture:** A new `cmd/deps.go` file holds all dependency logic. `ensureDeps()` is called from `rootCmd.PersistentPreRunE` before every command. It checks `exec.LookPath` for each tool and runs the appropriate platform installer if either is missing. Install commands stream output directly to the terminal.

**Tech Stack:** Go stdlib (`os/exec`, `runtime`), Cobra `PersistentPreRunE`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `cmd/deps.go` | Create | All dependency logic: helpers, platform installers, `ensureDeps` |
| `cmd/deps_test.go` | Create | Unit tests (helpers, platform installers, orchestrator) |
| `cmd/root.go` | Modify | Add `PersistentPreRunE` hook |
| `README.md` | Modify | Remove manual install instructions from Requirements section |

---

## Task 1: Skeleton + helpers

**Files:**
- Create: `cmd/deps.go`
- Create: `cmd/deps_test.go`

- [ ] **Step 1: Write failing tests for `toolExists` and `findPip`**

Create `cmd/deps_test.go`:

```go
package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// mockLookPath returns a LookPath func that finds only the named tools.
func mockLookPath(present ...string) func(string) (string, error) {
	set := make(map[string]bool, len(present))
	for _, p := range present {
		set[p] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", fmt.Errorf("%s: not found", name)
	}
}

func TestToolExists_found(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath("ffmpeg")
	if !toolExists("ffmpeg") {
		t.Error("expected true for present tool")
	}
}

func TestToolExists_notFound(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath()
	if toolExists("ffmpeg") {
		t.Error("expected false for absent tool")
	}
}

func TestFindPip_pip3(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath("pip3")
	if got := findPip(); got != "pip3" {
		t.Errorf("want pip3, got %q", got)
	}
}

func TestFindPip_pipFallback(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath("pip")
	if got := findPip(); got != "pip" {
		t.Errorf("want pip, got %q", got)
	}
}

func TestFindPip_python3Fallback(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath("python3")
	if got := findPip(); got != "python3 -m pip" {
		t.Errorf("want 'python3 -m pip', got %q", got)
	}
}

func TestFindPip_none(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath()
	if got := findPip(); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestFallbackInstructions_content(t *testing.T) {
	s := fallbackInstructions()
	for _, want := range []string{"ffmpeg", "openai-whisper", "brew", "apt-get", "winget", "pip3"} {
		if !strings.Contains(s, want) {
			t.Errorf("fallbackInstructions missing %q", want)
		}
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./cmd/ -run 'TestToolExists|TestFindPip|TestFallbackInstructions' -v
```
Expected: `FAIL` — `osLookPath`, `toolExists`, `findPip`, `fallbackInstructions` undefined.

- [ ] **Step 3: Create `cmd/deps.go` with helpers**

```go
package cmd

import (
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

// findPip returns the first available pip command, or "" if none found.
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

// cmdString joins name + args into a single display string (for error messages).
func cmdString(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./cmd/ -run 'TestToolExists|TestFindPip|TestFallbackInstructions' -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/deps.go cmd/deps_test.go
git commit -m "feat(deps): add toolExists, findPip, fallbackInstructions helpers"
```

---

## Task 2: `runCmdReal` + `installWhisperViaPip`

**Files:**
- Modify: `cmd/deps.go`
- Modify: `cmd/deps_test.go`

- [ ] **Step 1: Write failing tests for `installWhisperViaPip`**

Add to `cmd/deps_test.go`:

```go
func TestInstallWhisperViaPip_pip3Success(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("pip3")
	var got []string
	osRunCmd = func(name string, args ...string) error {
		got = append([]string{name}, args...)
		return nil
	}

	if err := installWhisperViaPip(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"pip3", "install", "openai-whisper"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestInstallWhisperViaPip_python3Fallback(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("python3")
	var got []string
	osRunCmd = func(name string, args ...string) error {
		got = append([]string{name}, args...)
		return nil
	}

	if err := installWhisperViaPip(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"python3", "-m", "pip", "install", "openai-whisper"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestInstallWhisperViaPip_noPip(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath()

	err := installWhisperViaPip()
	if err == nil || !strings.Contains(err.Error(), "no pip found") {
		t.Errorf("expected 'no pip found' error, got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./cmd/ -run 'TestInstallWhisperViaPip' -v
```
Expected: FAIL — `runCmdReal`, `installWhisperViaPip` undefined.

- [ ] **Step 3: Add `runCmdReal` and `installWhisperViaPip` to `cmd/deps.go`**

```go
import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

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
```

> **Note:** Update the `var` block at the top of `cmd/deps.go` — `osRunCmd` references `runCmdReal` which is now defined. The existing `var` block already has `osRunCmd = runCmdReal`; just ensure the import block adds `"fmt"`, `"log"`, and `"os"`.

- [ ] **Step 4: Run tests**

```bash
go test ./cmd/ -run 'TestInstallWhisperViaPip' -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/deps.go cmd/deps_test.go
git commit -m "feat(deps): add runCmdReal and installWhisperViaPip"
```

---

## Task 3: `installDarwin`

**Files:**
- Modify: `cmd/deps.go`
- Modify: `cmd/deps_test.go`

- [ ] **Step 1: Write failing tests**

Add to `cmd/deps_test.go`:

```go
func TestInstallDarwin_brewPresent_ffmpegOnly(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("brew", "pip3") // brew present, ffmpeg absent
	var cmds []string
	osRunCmd = func(name string, args ...string) error {
		cmds = append(cmds, cmdString(name, args))
		return nil
	}

	if err := installDarwin(true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 || cmds[0] != "brew install ffmpeg" {
		t.Errorf("want [brew install ffmpeg], got %v", cmds)
	}
}

func TestInstallDarwin_brewMissing_installsBrewThenFfmpeg(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("pip3") // no brew, no ffmpeg
	var cmds []string
	osRunCmd = func(name string, args ...string) error {
		cmds = append(cmds, cmdString(name, args))
		return nil
	}

	if err := installDarwin(true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) < 2 {
		t.Fatalf("expected ≥2 commands, got %v", cmds)
	}
	if !strings.Contains(cmds[0], "Homebrew") && !strings.Contains(cmds[0], "/bin/sh") {
		t.Errorf("first command should be homebrew install script, got: %q", cmds[0])
	}
	if cmds[1] != "brew install ffmpeg" {
		t.Errorf("second command should be brew install ffmpeg, got: %q", cmds[1])
	}
}

func TestInstallDarwin_whisperOnly(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("pip3")
	var cmds []string
	osRunCmd = func(name string, args ...string) error {
		cmds = append(cmds, cmdString(name, args))
		return nil
	}

	if err := installDarwin(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 || cmds[0] != "pip3 install openai-whisper" {
		t.Errorf("want [pip3 install openai-whisper], got %v", cmds)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./cmd/ -run 'TestInstallDarwin' -v
```
Expected: FAIL — `installDarwin` undefined.

- [ ] **Step 3: Add `installDarwin` to `cmd/deps.go`**

```go
// installDarwin installs missing deps on macOS.
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./cmd/ -run 'TestInstallDarwin' -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/deps.go cmd/deps_test.go
git commit -m "feat(deps): add installDarwin"
```

---

## Task 4: `installLinux`

**Files:**
- Modify: `cmd/deps.go`
- Modify: `cmd/deps_test.go`

- [ ] **Step 1: Write failing tests**

Add to `cmd/deps_test.go`:

```go
func TestInstallLinux_aptGet(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("apt-get", "pip3")
	var cmds []string
	osRunCmd = func(name string, args ...string) error {
		cmds = append(cmds, cmdString(name, args))
		return nil
	}

	if err := installLinux(true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 || cmds[0] != "apt-get install -y ffmpeg" {
		t.Errorf("want [apt-get install -y ffmpeg], got %v", cmds)
	}
}

func TestInstallLinux_aptFallback(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("apt", "pip3") // apt-get absent, apt present
	var cmds []string
	osRunCmd = func(name string, args ...string) error {
		cmds = append(cmds, cmdString(name, args))
		return nil
	}

	if err := installLinux(true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 || cmds[0] != "apt install -y ffmpeg" {
		t.Errorf("want [apt install -y ffmpeg], got %v", cmds)
	}
}

func TestInstallLinux_noApt(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath()

	err := installLinux(true, false)
	if err == nil || !strings.Contains(err.Error(), "apt-get or apt") {
		t.Errorf("expected apt-get/apt error, got: %v", err)
	}
}

func TestInstallLinux_whisperOnly(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("pip3")
	var cmds []string
	osRunCmd = func(name string, args ...string) error {
		cmds = append(cmds, cmdString(name, args))
		return nil
	}

	if err := installLinux(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 || cmds[0] != "pip3 install openai-whisper" {
		t.Errorf("want [pip3 install openai-whisper], got %v", cmds)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./cmd/ -run 'TestInstallLinux' -v
```
Expected: FAIL — `installLinux` undefined.

- [ ] **Step 3: Add `installLinux` to `cmd/deps.go`**

```go
// installLinux installs missing deps on Linux.
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./cmd/ -run 'TestInstallLinux' -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/deps.go cmd/deps_test.go
git commit -m "feat(deps): add installLinux"
```

---

## Task 5: `installWindows`

**Files:**
- Modify: `cmd/deps.go`
- Modify: `cmd/deps_test.go`

- [ ] **Step 1: Write failing tests**

Add to `cmd/deps_test.go`:

```go
func TestInstallWindows_winget(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("winget", "pip3")
	var cmds []string
	osRunCmd = func(name string, args ...string) error {
		cmds = append(cmds, cmdString(name, args))
		return nil
	}

	if err := installWindows(true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 || cmds[0] != "winget install --id Gyan.FFmpeg -e" {
		t.Errorf("want [winget install --id Gyan.FFmpeg -e], got %v", cmds)
	}
}

func TestInstallWindows_noWinget(t *testing.T) {
	orig := osLookPath
	defer func() { osLookPath = orig }()
	osLookPath = mockLookPath()

	err := installWindows(true, false)
	if err == nil || !strings.Contains(err.Error(), "winget not found") {
		t.Errorf("expected winget error, got: %v", err)
	}
}

func TestInstallWindows_whisperOnly(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("pip3")
	var cmds []string
	osRunCmd = func(name string, args ...string) error {
		cmds = append(cmds, cmdString(name, args))
		return nil
	}

	if err := installWindows(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 || cmds[0] != "pip3 install openai-whisper" {
		t.Errorf("want [pip3 install openai-whisper], got %v", cmds)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./cmd/ -run 'TestInstallWindows' -v
```
Expected: FAIL — `installWindows` undefined.

- [ ] **Step 3: Add `installWindows` to `cmd/deps.go`**

```go
// installWindows installs missing deps on Windows (best-effort via winget).
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./cmd/ -run 'TestInstallWindows' -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/deps.go cmd/deps_test.go
git commit -m "feat(deps): add installWindows"
```

---

## Task 6: `ensureDeps` orchestrator

**Files:**
- Modify: `cmd/deps.go`
- Modify: `cmd/deps_test.go`

- [ ] **Step 1: Write failing tests**

Add to `cmd/deps_test.go`:

```go
func TestEnsureDeps_allPresent_noInstall(t *testing.T) {
	origLook := osLookPath
	origRun := osRunCmd
	defer func() { osLookPath = origLook; osRunCmd = origRun }()

	osLookPath = mockLookPath("ffmpeg", "whisper")
	called := false
	osRunCmd = func(name string, args ...string) error {
		called = true
		return nil
	}

	if err := ensureDeps(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected no install commands when all deps present")
	}
}

func TestEnsureDeps_unknownOS_returnsError(t *testing.T) {
	origLook := osLookPath
	origGOOS := currentGOOS
	defer func() { osLookPath = origLook; currentGOOS = origGOOS }()

	osLookPath = mockLookPath() // nothing present
	currentGOOS = "plan9"

	err := ensureDeps()
	if err == nil || !strings.Contains(err.Error(), "auto-install not supported") {
		t.Errorf("expected unsupported OS error, got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./cmd/ -run 'TestEnsureDeps' -v
```
Expected: FAIL — `ensureDeps`, `currentGOOS` undefined.

- [ ] **Step 3: Add `ensureDeps` and `currentGOOS` to `cmd/deps.go`**

```go
import "runtime"

// currentGOOS is a var so tests can override the platform.
var currentGOOS = runtime.GOOS

// ensureDeps checks for ffmpeg and whisper, installing either if absent.
func ensureDeps() error {
	ffmpegMissing := !toolExists("ffmpeg")
	whisperMissing := !toolExists("whisper")

	if !ffmpegMissing && !whisperMissing {
		return nil
	}

	switch currentGOOS {
	case "darwin":
		return installDarwin(ffmpegMissing, whisperMissing)
	case "linux":
		return installLinux(ffmpegMissing, whisperMissing)
	case "windows":
		return installWindows(ffmpegMissing, whisperMissing)
	default:
		return fmt.Errorf("auto-install not supported on %s\n%s", currentGOOS, fallbackInstructions())
	}
}
```

- [ ] **Step 4: Run all deps tests**

```bash
go test ./cmd/ -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/deps.go cmd/deps_test.go
git commit -m "feat(deps): add ensureDeps orchestrator"
```

---

## Task 7: Wire into `root.go`

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Add `PersistentPreRunE` to `rootCmd` in `cmd/root.go`**

Replace the `var rootCmd` declaration with:

```go
var rootCmd = &cobra.Command{
	Use:     "whisperbatch",
	Version: version,
	Short:   "Batch transcribe audio files using OpenAI Whisper",
	Long: `whisperbatch transcribes a folder of audio files in parallel using the Whisper CLI.

Audio formats supported: mp3, wav, m4a, flac, ogg, mp4, webm
Output formats supported: txt, json, srt, vtt, tsv

Example:
  whisperbatch -i ./recordings
  whisperbatch -i ./recordings -o ./output -f txt -f srt -f json
  whisperbatch -i ./recordings -m large -w 8 --overwrite`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureDeps()
	},
	RunE: run,
}
```

- [ ] **Step 2: Build to confirm it compiles**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Run full test suite**

```bash
go test -race ./...
```
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat(deps): wire ensureDeps into PersistentPreRunE"
```

---

## Task 8: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Remove manual install prerequisite instructions**

In the `## Requirements` section, replace the install instructions block with a note that dependencies are handled automatically:

```markdown
## Requirements

| Requirement | Notes |
|-------------|-------|
| Go 1.22+ | Only needed for `go install` / building from source |
| Python 3.9+ | For the `whisper` CLI — installed automatically if missing |
| `whisper` on `$PATH` | Installed automatically on first run |
| `ffmpeg` on `$PATH` | Installed automatically on first run |

`whisperbatch` installs missing dependencies automatically on first run using
Homebrew (macOS), apt (Linux), or winget (Windows). No manual setup needed.
```

Remove the entire `Install Whisper and ffmpeg:` code block and the `faster-whisper` note beneath it (the note about multiple `--output_format` flags is no longer accurate since we removed that flag from the whisper call).

- [ ] **Step 2: Build and test one more time**

```bash
go build ./... && go test -race ./...
```
Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update README — deps are now auto-installed"
```

---

## Task 9: Final tag and push

- [ ] **Step 1: Push all commits**

```bash
git push origin main
```

- [ ] **Step 2: Tag and push — GoReleaser handles the release**

```bash
git tag v0.3.0
git push origin v0.3.0
```

Expected: GitHub Actions release workflow triggers, builds binaries and Docker images, creates the GitHub release automatically.
