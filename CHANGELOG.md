# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Built-in `default` and `solarized-gruvbox` color themes, selectable by CLI flag, `MUX_THEME`, or XDG configuration.
- Configurable, context-aware TUI keybindings with conflict validation and an on-demand `?` help card.
- Fullscreen live-preview task switcher with a one-row contextual session/window/pane title, compact centered selector, and bottom-left preview framing across every remaining row.
- Window movement between sessions, including an explicit warning when moving the final window removes the source session.
- Hierarchical session, window, and pane selection (#14):
  - `Tab` / `→` / `l` drills into the selected session or window
  - `Shift+Tab` / `←` / `h` returns to the parent level
  - The fullscreen preview follows the highlighted target via `tmux capture-pane -t session:window.pane`
  - `Enter` or `Backspace` on a window or pane attaches and focuses that exact target (`select-window` + `select-pane` before attach)
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
- Session ordering now behaves like an OS switcher: the invoking session is displayed first, the previous session is displayed second and initially highlighted, and older sessions follow in MRU order; background output no longer changes recency.
- Popup launch now uses a borderless `100% × 100%` canvas and generated keybindings preserve the invoking tmux session for switcher ordering.
- `Session.Windows` (int) split into `Session.WindowCount` (int) + `Session.Windows` ([]Window) — the latter is lazily populated on demand
- `AttachToSession(name)` signature extended to `AttachToSession(name, windowIdx, paneIdx)` — pass `-1` to keep tmux defaults
- Cross-platform `shortenPath` using `os.UserHomeDir()` instead of hardcoded `/Users/`
- Go version in go.mod updated to stable release

### Removed
- Vendor-specific process detection, badges, token and cost tracking, theme fields, and filesystem integrations. Generic tmux command display, previews, and Git branch/worktree metadata remain.
- The `mux status` command and its statusbar integration.
- Exported vendor-integration APIs and fields: `AITool`, `IsAICommand`, `LookupAITool`, `TokenUsage`, `FindClaudeSession`, `LoadTokenUsage`, `FormatTokens`, `Session.PanePID`, and `Theme.AITools`. Consumers should use `Session.ActiveCommand`, which now contains tmux's unmodified `pane_current_command` value.

### Fixed
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
