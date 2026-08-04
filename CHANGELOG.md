# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Multi-window/pane tree expansion in the session list (#14):
  - `Tab` / `→` / `l` to expand a session into its windows, then a window into its panes
  - `Shift+Tab` / `←` / `h` to collapse one level
  - Preview panel now follows the cursor — captures the targeted window or pane via `tmux capture-pane -t session:window.pane`
  - `Enter` on a window/pane row attaches and focuses that exact pane (`select-window` + `select-pane` before attach)
- `tmux.ListWindows` / `tmux.ListPanes` / `tmux.CapturePaneTarget` helpers
- MIT License
- English README with Korean translation (README.ko.md)
- CONTRIBUTING.md guide
- CODE_OF_CONDUCT.md (Contributor Covenant v2.1)
- Makefile with build, test, install, clean targets
- goreleaser configuration for automated releases
- VHS demo tape for recording demo GIFs
- Unit tests for tmux and UI packages
- `--version` flag
- `scripts/test-fixture.sh` for spinning up test sessions with multiple windows/panes

### Changed
- `Session.Windows` (int) split into `Session.WindowCount` (int) + `Session.Windows` ([]Window) — the latter is lazily populated on demand
- `AttachToSession(name)` signature extended to `AttachToSession(name, windowIdx, paneIdx)` — pass `-1` to keep tmux defaults
- Cross-platform `shortenPath` using `os.UserHomeDir()` instead of hardcoded `/Users/`
- Go version in go.mod updated to stable release

### Changed (API)
- `tmux.Session` gains an `ID` field (`#{session_id}`, e.g. `$5`) — the value to use for every `-t` target
- `tmux.KillSession` / `tmux.RenameSession` / `tmux.ListWindows` / `tmux.ListPanes` now expect `Session.ID` (a tmux target) instead of the session name
- `ui.Model.AttachName` now returns the tmux session ID rather than the display name (still passed straight to `AttachToSession`)

### Fixed
- Sessions whose names contain tmux target-syntax characters (`.` `:` `$` `=`, e.g. `v1.2`) could be created but never killed, renamed, previewed, or attached from mux — all `-t` targeting now uses the immutable session ID instead of the name
- TUI crash (`index out of range`) when the preview panel was 8 rows or shorter with a Claude token line visible (small terminals and `mux popup` on short clients)
- Token/cost readout was permanently blank for any session whose working directory contains a dot (`.worktrees/…`, `~/.config/…`) — Claude Code encodes `.` as `-` in its projects directory and `encodePath` now matches
- Token totals were silently undercounted when a session log line exceeded 1MB; the scanner cap is now 16MB and scan errors are surfaced instead of caching a partial sum
- Killing the last session (or the tmux server exiting) left a stale, phantom session list on screen; an empty listing now clears the list
- Failed kill/create/rename operations now show tmux's actual stderr message (e.g. `duplicate session: dev`) instead of a bare `exit status 1`, and kill failures are rendered in the UI instead of being silently discarded
- `install.sh` exited with code 1 (and leaked its temp directory) after a successful GitHub-release install, due to an EXIT trap referencing an out-of-scope `local` under `set -u`
- `renderPreview` test call missing `captured` parameter
- `setup-keybind` no longer corrupts `~/.tmux.conf` for [oh-my-tmux](https://github.com/gpakosz/.tmux) users (#15). Detects oh-my-tmux via symlink target or signature line, routes the bind line to `~/.tmux.conf.local` before the `# "$@"` sentinel, and cleans up any prior corrupt entry (including legacy untagged binds from older `install.sh`) from the main conf. `install.sh`'s shell fallback received the same treatment.

## [0.1.0] - 2026-03-30

### Added
- TUI session manager with list and live preview panels
- Real-time terminal output preview (500ms refresh)
- AI CLI detection (claude, codex, aider, gemini) with badge display
- Session create, delete, and rename from within the TUI
- Quick filter with `/` key
- Instant attach / switch-client
- Popup mode (`mux popup`) as tmux floating overlay
- `mux setup-keybind` for one-command tmux keybinding setup
- Cross-platform AI CLI detection (Linux/macOS)
