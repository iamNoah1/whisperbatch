# whisperbatch

[![CI](https://img.shields.io/github/actions/workflow/status/iamNoah1/whisperbatch/ci.yml?branch=main&label=CI)](https://github.com/iamNoah1/whisperbatch/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/iamNoah1/whisperbatch)](https://github.com/iamNoah1/whisperbatch/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/iamNoah1/whisperbatch)](https://goreportcard.com/report/github.com/iamNoah1/whisperbatch)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/iamNoah1/whisperbatch)](go.mod)

A fast, parallel CLI for batch-transcribing audio files with [OpenAI Whisper](https://github.com/openai/whisper).  
Drop a folder of recordings in — get transcripts out. Automatically selects the best model for your hardware.

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  WhisperBatch — Done
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Files processed : 24
  Succeeded       : 24
  Failed          : 0
  Total time      : 8m14s
  Model used      : medium (auto)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## Features

- **Parallel processing** — configurable worker pool, defaults to your CPU count
- **Auto model selection** — queries `nvidia-smi` for VRAM, falls back to RAM
- **Multiple output formats** — `txt`, `srt`, `vtt`, `json`, `tsv` in one pass
- **Safe by default** — never overwrites existing transcripts unless `--overwrite`
- **Graceful failures** — one bad file doesn't abort the batch
- **Progress bar** — live feedback with per-file failure reporting

---

## Requirements

| Requirement | Notes |
|-------------|-------|
| Go 1.22+ | Only needed for `go install` / building from source |
| Python 3.9+ | For the `whisper` CLI |
| `whisper` on `$PATH` | See install instructions below |
| `ffmpeg` on `$PATH` | Required by whisper for audio decoding |

Install Whisper and ffmpeg:

```bash
# macOS
brew install ffmpeg
pip install openai-whisper

# Ubuntu / Debian
sudo apt install ffmpeg
pip install openai-whisper
```

> **faster-whisper users:** `pip install faster-whisper` works too and natively supports
> multiple `--output_format` flags in a single run.

---

## Installation

### go install (recommended)

```bash
go install github.com/iamNoah1/whisperbatch@latest
```

### Pre-built binaries

Download the binary for your platform from the [Releases page](https://github.com/iamNoah1/whisperbatch/releases/latest), extract, and place on your `$PATH`.

```bash
# Example: Linux amd64
curl -L https://github.com/iamNoah1/whisperbatch/releases/latest/download/whisperbatch_linux_amd64.tar.gz \
  | tar -xz -C /usr/local/bin
```

### Docker

```bash
# Pull the image (includes Python, ffmpeg, and openai-whisper)
docker pull ghcr.io/iamnoah1/whisperbatch:latest

# Transcribe a folder
docker run --rm \
  -v /path/to/audio:/input:ro \
  -v /path/to/output:/output \
  ghcr.io/iamnoah1/whisperbatch -i /input -o /output
```

### Build from source

```bash
git clone https://github.com/iamNoah1/whisperbatch.git
cd whisperbatch
make build        # → ./whisperbatch
make install      # → $GOPATH/bin/whisperbatch
```

---

## Usage

```bash
# Transcribe all audio in ./recordings → txt (default)
whisperbatch -i ./recordings

# Multiple output formats, custom output folder
whisperbatch -i ./recordings -o ./output -f txt -f srt -f json

# Force a specific model, 8 workers, overwrite existing transcripts
whisperbatch -i ./recordings -m large -w 8 --overwrite

# Check version
whisperbatch --version
```

---

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--input` | `-i` | string | **required** | Folder containing audio files |
| `--output` | `-o` | string | same as input | Folder for output files |
| `--format` | `-f` | `[]string` | `txt` | Output format — repeatable: `txt json srt vtt tsv` |
| `--workers` | `-w` | int | CPU count | Parallel transcription workers |
| `--model` | `-m` | string | auto | Whisper model override: `tiny base medium large` |
| `--overwrite` | | bool | `false` | Overwrite existing output files |
| `--version` | | | | Print version and exit |

---

## Auto Model Selection

When `--model` is not set, `whisperbatch` picks the best model for your hardware:

| Resource available | Model |
|--------------------|-------|
| VRAM ≥ 10 GB **or** RAM ≥ 16 GB | `large` |
| VRAM ≥ 5 GB **or** RAM ≥ 8 GB | `medium` |
| VRAM ≥ 2 GB **or** RAM ≥ 4 GB | `base` |
| Less than the above | `tiny` |

GPU VRAM is detected via `nvidia-smi`. If that fails (no GPU or no driver),
available system RAM is used instead. The selection is logged at startup:

```
2025/01/15 10:23:01 model: medium (VRAM 6144 MB >= 5 GB)
```

---

## Supported Audio Formats

`.mp3` `.wav` `.m4a` `.flac` `.ogg` `.mp4` `.webm`

Files are discovered **recursively** from the input folder.

---

## Output File Naming

Outputs land in `--output` (defaults to the input folder), named after the source file stem:

```
/recordings/interview.mp3  →  <output>/interview.txt
                           →  <output>/interview.srt
                           →  <output>/interview.json
```

If all output files for a given input already exist and `--overwrite` is not set,
that file is skipped with a log message.

---

## GPU (CUDA) Docker Image

The default Docker image uses CPU-only Whisper. For GPU acceleration, build with a CUDA base:

```dockerfile
FROM nvidia/cuda:12.3.1-runtime-ubuntu22.04
RUN apt-get update && apt-get install -y python3 python3-pip ffmpeg
RUN pip3 install openai-whisper
COPY whisperbatch /usr/local/bin/whisperbatch
ENTRYPOINT ["whisperbatch"]
```

Then run with `--gpus all`:

```bash
docker run --gpus all --rm \
  -v /audio:/input:ro -v /output:/output \
  my-whisper-gpu -i /input -o /output
```

---

## Development

```bash
make build        # compile
make test         # run tests
make vet          # go vet
make lint         # golangci-lint (requires golangci-lint installed)
make release-dry  # goreleaser snapshot (requires goreleaser installed)
make clean        # remove artifacts
```

---

## Releasing

Push a semver tag — the release workflow handles everything:

```bash
git tag v1.2.3
git push origin v1.2.3
```

GitHub Actions will:
1. Build binaries for Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)
2. Build and push multi-arch Docker images to GHCR
3. Create a GitHub Release with a changelog and `checksums.txt`

---

## Project Structure

```
whisperbatch/
├── main.go                   Entry point
├── cmd/root.go               Cobra command & flag definitions
├── transcriber/
│   ├── transcriber.go        Worker pool orchestration
│   ├── whisper.go            Subprocess call to whisper CLI
│   └── model.go              Auto model selection (VRAM / RAM)
├── fileutil/fileutil.go      Audio file discovery, output path helpers
├── Dockerfile                Self-contained image (Go build + runtime)
├── Dockerfile.release        Runtime-only image (used by GoReleaser)
├── .goreleaser.yaml          Cross-platform release configuration
└── .github/workflows/        CI and release pipelines
```

---

## License

[MIT](LICENSE) © iamNoah1
