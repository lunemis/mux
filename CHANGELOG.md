# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Built-in `default` and `solarized-gruvbox` color themes, selectable by CLI flag, `TMUX_PEEKER_THEME`, or XDG configuration.
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
- English README
- CONTRIBUTING.md guide
- CODE_OF_CONDUCT.md (Contributor Covenant v2.1)
- Makefile with build, test, install, clean targets
- goreleaser configuration for automated releases
- VHS demo tape for recording demo GIFs
- Unit tests for tmux and UI packages
- `--version` flag
- `scripts/test-fixture.sh` for spinning up test sessions with multiple windows/panes

### Changed
- New Session now asks only for a name and lets tmux choose the normal starting directory.
- Selector and help cards now use an opaque semantic `surface` color across titles, borders, normal rows, padding, and blank cells while preserving the selected-row and terminal-canvas backgrounds.
- Renamed the independent fork to `tmux-peeker`, including its binary, Go module, config directory, environment variables, tmux marker, installer, release metadata, and documentation. The new identity intentionally provides no `mux` executable alias or automatic upstream-state migration.
- Session ordering now behaves like an OS switcher: the invoking session is displayed first, the previous session is displayed second and initially highlighted, and older sessions follow in MRU order; background output no longer changes recency.
- Popup launch now uses a borderless `100% × 100%` canvas and generated keybindings preserve the invoking tmux session for switcher ordering.
- `Session.Windows` (int) split into `Session.WindowCount` (int) + `Session.Windows` ([]Window) — the latter is lazily populated on demand
- `AttachToSession(name)` signature extended to `AttachToSession(name, windowIdx, paneIdx)` — pass `-1` to keep tmux defaults
- Cross-platform `shortenPath` using `os.UserHomeDir()` instead of hardcoded `/Users/`
- Go version in go.mod updated to stable release

### Removed
- The unused New Session directory field, its `create.switch_field` keybinding, and `CreateSessionWithDir` helper.
- Vendor-specific process detection, badges, token and cost tracking, theme fields, and filesystem integrations. Generic tmux command display, previews, and Git branch/worktree metadata remain.
- The `mux status` command and its statusbar integration.
- Exported vendor-integration APIs and fields: `AITool`, `IsAICommand`, `LookupAITool`, `TokenUsage`, `FindClaudeSession`, `LoadTokenUsage`, `FormatTokens`, `Session.PanePID`, and `Theme.AITools`. Consumers should use `Session.ActiveCommand`, which now contains tmux's unmodified `pane_current_command` value.

### Fixed
- Configured `space` bindings now match Bubble Tea's literal Space key representation, display visibly in help, and conflict correctly with literal `" "` bindings.
- `renderPreview` test call missing `captured` parameter
- `setup-keybind` routes its owned entry to `~/.tmux.conf.local` for [oh-my-tmux](https://github.com/gpakosz/.tmux) users (#15) and replaces only the `# tmux-peeker popup keybinding` marker. Upstream and user-owned bindings remain untouched. `install.sh`'s shell fallback follows the same ownership rule.

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
