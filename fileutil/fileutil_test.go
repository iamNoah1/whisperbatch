package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAudioFiles(t *testing.T) {
	dir := t.TempDir()

	create := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	create("a.mp3")
	create("b.wav")
	create("c.m4a")
	create("d.flac")
	create("e.ogg")
	create("f.mp4")
	create("g.webm")
	create("h.txt") // not audio
	create("i.pdf") // not audio
	create("j.MP3") // uppercase — should still match

	files, err := FindAudioFiles(dir)
	if err != nil {
		t.Fatalf("FindAudioFiles: %v", err)
	}

	if got, want := len(files), 8; got != want {
		t.Errorf("got %d files, want %d", got, want)
	}
}

func TestFindAudioFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := FindAudioFiles(dir)
	if err != nil {
		t.Fatalf("FindAudioFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files, got %d", len(files))
	}
}

func TestOutputPath(t *testing.T) {
	cases := []struct {
		source    string
		outputDir string
		format    string
		want      string
	}{
		{"/audio/my podcast.mp3", "/out", "txt", "/out/my podcast.txt"},
		{"/audio/episode.wav", "/out", "srt", "/out/episode.srt"},
		{"/audio/talk.mp3", "/out", "json", "/out/talk.json"},
	}
	for _, tc := range cases {
		got := OutputPath(tc.source, tc.outputDir, tc.format)
		if got != tc.want {
			t.Errorf("OutputPath(%q, %q, %q) = %q, want %q",
				tc.source, tc.outputDir, tc.format, got, tc.want)
		}
	}
}
