# mux

**A fast tmux session switcher with live previews.**

When work spans several tmux sessions, windows, and panes, finding the right terminal should not interrupt your flow. mux shows the selected target's live output and lets you switch in a keystroke.

[한국어](README.ko.md)

![Go](https://img.shields.io/badge/Go-1.24.2+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

## The Problem

tmux's built-in `choose-session` gives you names but not enough context to identify the terminal you need. You end up cycling through sessions, windows, and panes to find the right process.

## How mux solves it

### Fullscreen live preview — every window and pane
See the selected target's terminal output across the full mux canvas before you switch. A compact picker floats in the center while the preview follows your selection. Press `l`, `Right`, or `Tab` to drill from sessions into windows and panes; use `h`, `Left`, or `Shift+Tab` to return.

### Generic command display
Session and pane rows show the command reported directly by tmux, without process-specific integrations.

### Git branch & worktree display
Each session shows its current git branch. Linked worktrees are visually distinguished so you can tell at a glance which sessions are working on isolated branches.

### Popup overlay
Press one key to summon mux on top of whatever you're doing. Pick a session and you're there.

### Vim-style navigation
`j`/`k` to browse, `/` to filter, `Enter` or `Backspace` to attach. No mouse needed.

## Quick Start

```bash
# One-line interactive installer (recommended)
curl -sSL https://raw.githubusercontent.com/lunemis/mux/main/install.sh | bash

# Or install manually
brew install lunemis/tap/mux   # or: go install github.com/lunemis/mux/cmd/mux@latest
mux                             # launch the session manager
```

For the best experience, set up popup mode (opens mux as a floating overlay):

```bash
mux setup-keybind               # binds prefix + m and prints the reload command
```

Run the reload command it prints. Then press `Ctrl+b` followed by `m` anywhere in tmux to open mux.

## Installation

### Interactive installer (recommended)

The installer guides you through binary installation and keybinding setup:

```bash
curl -sSL https://raw.githubusercontent.com/lunemis/mux/main/install.sh | bash
```

### Homebrew

```bash
brew install lunemis/tap/mux
```

### From source

```bash
git clone https://github.com/lunemis/mux.git
cd mux
make install   # builds and installs to /usr/local/bin
```

### Go install

```bash
go install github.com/lunemis/mux/cmd/mux@latest
```

## Usage

### Basic

Run `mux` to open the session manager. Use `j`/`k` to navigate, `Enter` or `Backspace` to attach, and `q` to quit.

A one-row contextual title (`tmux session picker`, `tmux window picker`, or `tmux pane picker`) sits above the selected target's **live preview**, updated every 500ms. The preview fills every remaining row edge-to-edge with no outer border or separator. A compact picker floats in the center and shows one hierarchy level at a time. The preview remains anchored at the bottom-left so prompts and status rows stay visible. Press `?` to toggle contextual key help.

Sessions behave like an OS window switcher. Inside tmux, the current invoking session appears first, the previously used session appears second and is initially highlighted, and older sessions follow in MRU order. Pressing `Enter` or `Backspace` immediately switches to the highlighted previous session; reopening mux highlights the session you just left. Background output does not reorder the list. Outside tmux, the list remains MRU-first. Never-visited sessions fall back to newest creation time, then name.

mux-managed popup bindings created before this switcher behavior must be regenerated once so they pass the originating session into the popup. Run the reload command printed by `setup-keybind`:

```bash
mux setup-keybind
```

If you maintain a custom popup binding, make it invoke `mux popup` with the originating session instead of launching `mux` directly. For example, this global binding uses `Ctrl+Backspace`:

```tmux
bind-key -n C-BSpace run-shell 'MUX_ORIGIN_SESSION=#{q:session_name} "/absolute/path/to/mux" popup'
```

### Themes

mux includes two built-in color themes:

- `default` — the original dark-terminal palette
- `solarized-gruvbox` — a light theme inspired by Solarized contrast and Gruvbox Light Soft colors

Select a theme with `--theme`:

```bash
mux --theme solarized-gruvbox
mux --theme solarized-gruvbox popup
```

Persist the selection in `$XDG_CONFIG_HOME/mux/config.json` (or `~/.config/mux/config.json` when `XDG_CONFIG_HOME` is unset):

```bash
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/mux"
cat > "${XDG_CONFIG_HOME:-$HOME/.config}/mux/config.json" <<'EOF'
{
  "theme": "solarized-gruvbox"
}
EOF
```

Or set it for the current environment:

```bash
export MUX_THEME=solarized-gruvbox
mux
```

Theme precedence is `--theme`, then `MUX_THEME`, then the XDG config, then `default`.

Theme palettes live in [`theme/*.json`](theme/). To add a built-in theme, copy an existing file, give it a unique `name`, update its semantic UI colors, then rebuild mux. Set `colors.background` to `"NONE"` to preserve your terminal's background. The switcher selector's titled top edge uses `colors.separator` (`#2563EB` blue in the default theme) independently from its remaining `colors.border` edges. Theme files are embedded into the binary at build time.

### Custom keybindings

Every mux action can be rebound in the same XDG `config.json`. Overrides are partial: an action you omit keeps its default keys, while an action you include replaces its defaults. Each action accepts one or more keys.

```json
{
  "theme": "solarized-gruvbox",
  "keybindings": {
    "global": {
      "quit": ["ctrl+q"]
    },
    "list": {
      "up": ["w", "up"],
      "down": ["s", "down"],
      "create": ["c"]
    },
    "create": {
      "submit": ["ctrl+s"],
      "cancel": ["ctrl+x"]
    }
  }
}
```

Keys use Bubble Tea's case-sensitive names, such as `enter`, `backspace`, `esc`, `tab`, `shift+tab`, `up`, `right`, `ctrl+c`, or a literal character. mux rejects unknown contexts/actions, empty bindings, and keys assigned to conflicting actions in the same mode. The contextual help card and prompts show the active bindings.

| Context | Action | Default keys |
|---|---|---|
| `global` | `quit` | `ctrl+c` |
| `list` | `up` / `down` | `up`, `k` / `down`, `j` |
| `list` | `first` / `last` | `g` / `G` |
| `list` | `expand` / `collapse` | `tab`, `right`, `l` / `shift+tab`, `left`, `h` |
| `list` | `attach` | `enter`, `backspace` |
| `list` | `help` | `?` |
| `list` | `create` / `rename` / `kill` | `n` / `r` / `x` |
| `list` | `move_window` | `m` |
| `list` | `filter` / `clear_filter` | `/` / `esc` |
| `list` | `quit` | `q` |
| `create` | `switch_field` | `tab`, `shift+tab` |
| `create` | `submit` / `cancel` | `enter` / `esc` |
| `rename` | `submit` / `cancel` | `enter` / `esc` |
| `filter` | `apply` / `clear` | `enter` / `esc` |
| `kill` | `confirm` / `cancel` | `y`, `Y` / `any` |
| `move` | `up` / `down` | `up`, `k` / `down`, `j` |
| `move` | `confirm` / `cancel` | `enter` / `esc` |

`any` is a fallback reserved for `kill.cancel`; replace it with explicit keys such as `["n", "esc"]` if only those keys should cancel. These settings control keys inside the mux TUI. The external tmux popup binding remains configured separately with `mux setup-keybind`.

To move a window, drill into a session, select a window, and press `m`. Choose another session and press `Enter`; `Esc` cancels. mux preserves the destination's active window and uses its next free window index. Moving a session's final window is allowed; the chooser warns that tmux will remove the now-empty source session and may detach clients attached to it.

### Popup mode (recommended)

Open mux as a borderless fullscreen task switcher inside tmux, regardless of the program running in the foreground.

```bash
# Set up the keybinding (one-time)
mux setup-keybind          # prefix + m (default)
mux setup-keybind Space    # or use a different key
```

Run the reload command printed by `setup-keybind`. You can also open the popup manually with `mux popup`.

> **Note:** Popup mode requires tmux 3.2+

### Default keybindings

These defaults can be replaced through [custom keybindings](#custom-keybindings).

| Key | Action |
|---|---|
| `j` / `k` | Move down / up |
| `g` / `G` | Jump to first / last |
| `Tab` / `→` / `l` | Drill into session → windows → panes |
| `Shift+Tab` / `←` / `h` | Return to the parent level |
| `Enter` / `Backspace` | Attach (focuses the selected window/pane) |
| `?` | Toggle contextual help |
| `n` | Create new session |
| `r` | Rename session |
| `x` | Delete session (with confirmation) |
| `m` | Move the selected window to another session |
| `/` | Filter sessions by name or path |
| `Esc` | Clear filter / cancel |
| `q` | Quit |

## Requirements

- tmux (popup mode requires 3.2+)
- Linux or macOS

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
