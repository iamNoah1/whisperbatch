# ─── Stage 1: build the Go binary ─────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Cache dependency downloads separately from source code.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
        -ldflags="-s -w" \
        -o whisperbatch .

# ─── Stage 2: runtime with Python + Whisper + ffmpeg ──────────────────────────
FROM python:3.12-slim

LABEL org.opencontainers.image.title="whisperbatch" \
      org.opencontainers.image.description="Batch audio transcription via OpenAI Whisper" \
      org.opencontainers.image.source="https://github.com/iamNoah1/whisperbatch" \
      org.opencontainers.image.licenses="MIT"

# ffmpeg is required by whisper for audio decoding.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg \
 && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir openai-whisper

COPY --from=builder /app/whisperbatch /usr/local/bin/whisperbatch

# Mount your audio files at /input and outputs will appear at /output.
# Example:
#   docker run --rm -v /host/audio:/input -v /host/out:/output \
#     ghcr.io/iamnoah1/whisperbatch -i /input -o /output
ENTRYPOINT ["whisperbatch"]
