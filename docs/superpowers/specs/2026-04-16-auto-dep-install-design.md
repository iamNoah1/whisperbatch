# Auto-install Dependencies — Design

**Date:** 2026-04-16  
**Project:** whisperbatch  
**Status:** Approved

---

## Goal

Users should be able to run `whisperbatch` without manually installing `ffmpeg` or `openai-whisper` first. If either dependency is missing, whisperbatch installs it automatically and continues.

---

## Architecture

A single new file `cmd/deps.go` in the existing `cmd` package.

`rootCmd` gains a `PersistentPreRunE` hook that runs before every command:

```go
rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    return ensureDeps()
}
```

`ensureDeps()` calls `exec.LookPath` for `ffmpeg` and `whisper`. If both exist it returns immediately (nanoseconds, no user-visible overhead). If either is missing it logs what it is about to do, runs the appropriate installer with output streamed to the terminal, and returns an error only if the install fails. On success the main command continues normally.

---

## Per-platform Install Logic

All install commands stream stdout/stderr directly to the terminal so the user sees real-time progress.

### macOS
| Dependency | Install steps |
|-----------|---------------|
| `ffmpeg`  | Check for `brew`; if missing, run the official Homebrew install script (`https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh`) via `/bin/bash -c "$(curl -fsSL ...)"`. Then `brew install ffmpeg`. |
| `whisper` | `pip3 install openai-whisper` → `pip install openai-whisper` → `python3 -m pip install openai-whisper` → fallback instructions |

### Linux
| Dependency | Install steps |
|-----------|---------------|
| `ffmpeg`  | `apt-get install -y ffmpeg` → `apt install -y ffmpeg` → fallback instructions. Both commands run without an explicit `sudo` prefix — the subprocess inherits the terminal so a sudo password prompt will appear naturally if required. |
| `whisper` | pip fallback chain (same as macOS) |

### Windows
| Dependency | Install steps |
|-----------|---------------|
| `ffmpeg`  | `winget install --id Gyan.FFmpeg -e` → fallback instructions |
| `whisper` | pip fallback chain (same as macOS) |

**pip fallback chain (all platforms):** `pip3` → `pip` → `python3 -m pip` → print instructions

---

## User Feedback

Before each install step, a log line is printed:

```
2026/04/16 10:23:01 ffmpeg not found — installing via Homebrew
2026/04/16 10:23:15 whisper not found — installing via pip3
```

Install tool output (brew progress bars, pip download output, apt logs) streams directly to the terminal so the user can see what is happening.

---

## Error Handling

If any install step fails, whisperbatch prints platform-appropriate manual instructions and exits with a non-zero status:

```
could not install ffmpeg automatically.
Please install it manually:
  macOS:   brew install ffmpeg
  Linux:   sudo apt-get install ffmpeg
  Windows: winget install --id Gyan.FFmpeg -e
```

If ffmpeg installs successfully but whisper fails (or vice versa), whisperbatch still exits — a half-installed state would produce a confusing failure during transcription. The user reruns `whisperbatch` once the remaining dependency is resolved.

---

## Files Changed

| File | Change |
|------|--------|
| `cmd/deps.go` | New — `ensureDeps()` + platform install logic |
| `cmd/root.go` | Add `PersistentPreRunE` hook |
| `README.md` | Remove manual install instructions from Requirements section |

---

## Out of Scope

- Detecting/managing Python virtual environments
- Installing Python itself (pip assumed to exist alongside Python)
- Updating already-installed versions of ffmpeg or whisper
