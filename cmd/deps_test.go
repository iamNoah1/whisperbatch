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
