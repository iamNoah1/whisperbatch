package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/iamNoah1/whisperbatch/fileutil"
	"github.com/iamNoah1/whisperbatch/transcriber"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X github.com/iamNoah1/whisperbatch/cmd.version=<ver>".
var version = "dev"

var (
	inputDir  string
	outputDir string
	formats   []string
	workers   int
	model     string
	overwrite bool
)

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

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&inputDir, "input", "i", "", "Folder containing audio files (required)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Folder for output files (default: same as input)")
	rootCmd.Flags().StringArrayVarP(&formats, "format", "f", []string{"txt"}, "Output formats: txt, json, srt, vtt, tsv (repeatable)")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", 1, "Number of parallel transcription workers (default 1; each whisper process already uses all CPUs)")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "Override auto-selected Whisper model (tiny/base/medium/large)")
	rootCmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing output files")

	if err := rootCmd.MarkFlagRequired("input"); err != nil {
		log.Fatalf("failed to mark input flag as required: %v", err)
	}
}

func run(_ *cobra.Command, _ []string) error {
	if outputDir == "" {
		outputDir = inputDir
	}

	files, err := fileutil.FindAudioFiles(inputDir)
	if err != nil {
		return fmt.Errorf("discovering audio files: %w", err)
	}
	if len(files) == 0 {
		log.Printf("no audio files found in %s", inputDir)
		return nil
	}
	log.Printf("found %d audio file(s) in %s", len(files), inputDir)

	selectedModel := model
	modelAuto := model == ""
	if modelAuto {
		selectedModel = transcriber.SelectModel()
	} else {
		log.Printf("model: %s (from --model flag)", selectedModel)
	}

	cfg := transcriber.Config{
		OutputDir: outputDir,
		Formats:   formats,
		Workers:   workers,
		Model:     selectedModel,
		Overwrite: overwrite,
	}

	start := time.Now()
	summary := transcriber.RunBatch(files, cfg)
	summary.TotalWall = time.Since(start)

	printSummary(summary, selectedModel, modelAuto)

	if summary.Failed > 0 {
		os.Exit(1)
	}
	return nil
}

func printSummary(s transcriber.Summary, modelName string, auto bool) {
	modelLabel := modelName
	if auto {
		modelLabel += " (auto)"
	}

	const bar = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	fmt.Printf("\n%s\n", bar)
	fmt.Printf("  WhisperBatch — Done\n")
	fmt.Printf("%s\n", bar)
	fmt.Printf("  Files processed : %d\n", s.Total)
	fmt.Printf("  Succeeded       : %d\n", s.Succeeded)
	fmt.Printf("  Failed          : %d\n", s.Failed)
	fmt.Printf("  Total time      : %s\n", s.TotalWall.Round(time.Second))
	fmt.Printf("  Model used      : %s\n", modelLabel)
	fmt.Printf("%s\n", bar)
}
