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
		t.Fatalf("expected >=2 commands, got %v", cmds)
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
