package fileutil

import (
	"os"
	"path/filepath"
	"strings"
)

// audioExtensions is the set of file extensions considered audio/video files.
var audioExtensions = map[string]bool{
	".mp3":  true,
	".wav":  true,
	".m4a":  true,
	".flac": true,
	".ogg":  true,
	".mp4":  true,
	".webm": true,
}

// FindAudioFiles walks dir recursively and returns absolute paths of all audio files.
func FindAudioFiles(dir string) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if audioExtensions[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// OutputPath returns the expected output file path for a given source file, output directory,
// and whisper format (e.g. "txt", "srt", "json").
func OutputPath(sourceFile, outputDir, format string) string {
	base := filepath.Base(sourceFile)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(outputDir, stem+"."+format)
}
