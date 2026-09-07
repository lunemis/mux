# tmux-peeker

> **Peek before you switch.**

**tmux-peeker is a visual workspace switcher for tmux.** Preview sessions, windows, and panes before jumping into them.

## Why tmux-peeker?

A session name is rarely enough when several terminals look alike. tmux-peeker keeps the selected target's live output behind a compact picker, so you can recognize the workspace before switching.

- **Live previews** — see the selected session, window, or pane, refreshed every 500 ms
- **Hierarchy-aware navigation** — drill from sessions into windows and panes
- **Fast switching** — jump to the exact highlighted target with one key
- **MRU ordering** — move between recent workspaces like an operating-system switcher
- **Useful context** — see active commands and Git branch/worktree metadata
- **Workspace management** — create, rename, kill, filter, and move windows between sessions
- **Popup workflow** — open a borderless full-screen tmux popup from anywhere
- **Configurable controls and themes** — keep the workflow comfortable without vendor-specific integrations

## Install

### From source

Requires Go 1.24.2+ and tmux.

```bash
git clone https://github.com/aemonge/tmux-peeker.git
cd tmux-peeker
make test
make local-install
```

`make local-install` installs `tmux-peeker` into `~/.local/bin`. Ensure that directory is on `PATH`.

### Go

```bash
go install github.com/aemonge/tmux-peeker/cmd/tmux-peeker@latest
```

### Installer

```bash
curl -fsSL https://raw.githubusercontent.com/aemonge/tmux-peeker/main/install.sh | bash
```

The remote commands become available after the GitHub repository is published under the `tmux-peeker` name. No Homebrew tap is currently maintained.

## Quick start

```bash
tmux-peeker
```

Use `j`/`k` or the arrow keys to move, `Enter` or `Backspace` to attach, and `q` to quit.

### Navigate the hierarchy

| Action | Default keys |
|---|---|
| Move | `j` / `k`, `↓` / `↑` |
| First / last | `g` / `G` |
| Drill into session or window | `l`, `→`, `Tab` |
| Return to parent | `h`, `←`, `Shift+Tab` |
| Attach selected target | `Enter`, `Backspace` |
| Create / rename / kill | `n` / `r` / `x` |
| Move selected window | `m` |
| Filter / clear filter | `/` / `Esc` |
| Help | `?` |
| Quit | `q` |

Inside tmux, the invoking session stays first and the previously used session is highlighted next. Older sessions follow in MRU order. Background activity does not reorder them. Selecting a window or pane focuses that exact target before attaching.

## Popup mode

Let tmux-peeker install its owned popup binding:

```bash
tmux-peeker setup-keybind       # prefix + m
tmux-peeker setup-keybind Space # choose another tmux key
```

Run the reload command it prints. You can also open the popup directly:

```bash
tmux-peeker popup
```

A custom global binding must preserve the invoking session:

```tmux
bind-key -n C-BSpace run-shell 'TMUX_PEEKER_ORIGIN_SESSION=#{q:session_name} "/absolute/path/to/tmux-peeker" popup'
```

Generated lines carry the marker `# tmux-peeker popup keybinding`. Setup only replaces lines with that marker; bindings owned by `mux` or other tools remain untouched.

## Configuration

The config file is:

- `$XDG_CONFIG_HOME/tmux-peeker/config.json`, or
- `~/.config/tmux-peeker/config.json` when `XDG_CONFIG_HOME` is unset

Example:

```json
{
  "theme": "solarized-gruvbox",
  "keybindings": {
    "list": {
      "attach": ["enter", "space"],
      "up": ["k"],
      "down": ["j"],
      "quit": ["q", "esc"]
    },
    "create": {
      "cancel": ["esc", "ctrl+x"]
    }
  }
}
```

Overrides are partial: omitted actions retain their defaults, while a configured action replaces its default key list. Use Bubble Tea's case-sensitive names such as `enter`, `backspace`, `space`, `esc`, `tab`, `shift+tab`, `up`, `right`, and `ctrl+c`, or use a literal character. `space` is the visible alias for the Space key; a literal `" "` remains compatible. Conflicting keys are rejected.

Run `?` inside the picker to see active bindings. `any` is reserved for `kill.cancel` and means that any key cancels the confirmation.

### Themes

Built-in themes:

- `default`
- `solarized-gruvbox`

Select one for a run:

```bash
tmux-peeker --theme solarized-gruvbox
tmux-peeker --theme solarized-gruvbox popup
```

Or set:

```bash
export TMUX_PEEKER_THEME=solarized-gruvbox
```

Theme precedence is `--theme`, then `TMUX_PEEKER_THEME`, then the config file, then `default`. Palettes live in [`theme/*.json`](theme/) and are embedded at build time. Set `colors.background` to `"NONE"` to preserve the terminal canvas background. The required `colors.surface` role keeps selector and help cards opaque independently of that canvas setting.

## Migrating from `mux`

tmux-peeker owns a separate binary, config directory, environment namespace, and tmux marker. It never reads, moves, deletes, or rewrites upstream `mux` state.

Copy your selected settings explicitly:

```bash
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/tmux-peeker"
cp "${XDG_CONFIG_HOME:-$HOME/.config}/mux/config.json" \
  "${XDG_CONFIG_HOME:-$HOME/.config}/tmux-peeker/config.json"
```

Then replace any `MUX_THEME` usage with `TMUX_PEEKER_THEME` and run `tmux-peeker setup-keybind`. Existing `mux` binding text remains untouched. Because tmux activates only one command for a given key in a key table, choose a different key if both tools must remain usable; otherwise inspect your tmux configuration and remove the upstream binding manually when you no longer want it.

There is intentionally no permanent `mux` executable alias.

## Development

```bash
make test
make build
make local-install
```

Additional checks used for releases:

```bash
go vet ./...
go test -race ./...
shellcheck install.sh
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for project conventions.

## Independent-fork provenance

tmux-peeker is an independent fork derived from [`lunemis/mux`](https://github.com/lunemis/mux). It preserves the upstream Git history and MIT license while pursuing a vendor-neutral tmux workspace-switching product. It is not affiliated with or endorsed by the upstream project.

See [NOTICE](NOTICE) for attribution details.

## License

[MIT](LICENSE)
