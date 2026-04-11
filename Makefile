MODULE  := github.com/iamNoah1/whisperbatch
BINARY  := whisperbatch
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X $(MODULE)/cmd.version=$(VERSION)

.PHONY: build test vet lint install clean docker docker-run release-dry

## build: compile the binary for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: run all tests
test:
	go test -race -timeout 120s ./...

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (requires: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	golangci-lint run ./...

## install: install binary to $GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" .

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

## docker: build the Docker image (self-contained, includes whisper)
docker:
	docker build -t $(BINARY):$(VERSION) .

## docker-run: transcribe ./input → ./output inside Docker
docker-run:
	docker run --rm \
		-v "$(CURDIR)/input:/input:ro" \
		-v "$(CURDIR)/output:/output" \
		$(BINARY):$(VERSION) -i /input -o /output

## release-dry: dry-run GoReleaser (requires goreleaser installed)
release-dry:
	goreleaser release --snapshot --clean

.DEFAULT_GOAL := build

# Self-documenting help target
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
