# tmux-peeker README demo
#
# Validate without recording:
#   vhs validate demo.tape
#
# Record:
#   vhs demo.tape
#
# If recording is interrupted, clean the isolated server and workspace:
#   root="/tmp/tmux-peeker-vhs-${UID}"
#   tmux -S "$root/tmux.sock" kill-server 2>/dev/null || true
#   rm -rf "$root"

Output assets/demo.gif

Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 35ms

# Build into an isolated workspace before changing HOME. A private tmux wrapper
# keeps fixture sessions and captured content away from the user's live server.
Hide
Type "set -e"
Enter
Type 'export TMUX_PEEKER_REAL_TMUX=$(command -v tmux)'
Enter
Type 'export TMUX_PEEKER_DEMO_ROOT="/tmp/tmux-peeker-vhs-${UID}"'
Enter
Type 'mkdir "$TMUX_PEEKER_DEMO_ROOT"'
Enter
Type 'export TMUX_PEEKER_DEMO_SOCKET="$TMUX_PEEKER_DEMO_ROOT/tmux.sock"'
Enter
Type "trap '\$TMUX_PEEKER_REAL_TMUX -S \$TMUX_PEEKER_DEMO_SOCKET kill-server >/dev/null 2>&1 || true; rm -rf \$TMUX_PEEKER_DEMO_ROOT' EXIT INT TERM"
Enter
Type 'mkdir -p "$TMUX_PEEKER_DEMO_ROOT/bin" "$TMUX_PEEKER_DEMO_ROOT/home" "$TMUX_PEEKER_DEMO_ROOT/config"'
Enter
Type 'export XDG_CONFIG_HOME="$TMUX_PEEKER_DEMO_ROOT/config"'
Enter
Type 'make build BINARY="$TMUX_PEEKER_DEMO_ROOT/bin/tmux-peeker" >/dev/null'
Enter
Type 'printf "#!/bin/sh\nexec \"\$TMUX_PEEKER_REAL_TMUX\" -S \"\$TMUX_PEEKER_DEMO_SOCKET\" -f /dev/null \"\$@\"\n" > "$TMUX_PEEKER_DEMO_ROOT/bin/tmux"'
Enter
Type 'chmod +x "$TMUX_PEEKER_DEMO_ROOT/bin/tmux"'
Enter
Type 'export PATH="$TMUX_PEEKER_DEMO_ROOT/bin:$PATH"'
Enter
Type 'export HOME="$TMUX_PEEKER_DEMO_ROOT/home" SHELL=/bin/sh PS1="$ "'
Enter
Type "unset TMUX TMUX_PANE RPROMPT"
Enter
Type "scripts/test-fixture.sh up >/dev/null"
Enter
Type "clear"
Enter
Show
Sleep 500ms

# Start with the light palette so the opaque selector surface is unmistakable.
Type "tmux-peeker --theme solarized-gruvbox"
Sleep 300ms
Enter
Sleep 2s

# Narrow the session list to the multi-window fixture.
Type "/"
Sleep 300ms
Type@70ms "multi"
Sleep 300ms
Enter
Sleep 1s

# Drill from the session into its windows, select the server, then inspect panes.
Tab
Sleep 1200ms
Down
Sleep 1s
Tab
Sleep 1200ms
Down
Sleep 1s

# Show the opaque contextual help card, then return to the selected pane.
Type "?"
Sleep 2s
Escape
Sleep 1s

# Leave the picker without attaching so the recording remains deterministic.
Type "q"
Sleep 500ms

# Remove only the isolated server, fixtures, binary, HOME, and config.
Hide
Type '"$TMUX_PEEKER_REAL_TMUX" -S "$TMUX_PEEKER_DEMO_SOCKET" kill-server >/dev/null 2>&1 || true'
Enter
Type 'rm -rf "$TMUX_PEEKER_DEMO_ROOT"'
Enter
Type "trap - EXIT INT TERM"
Enter
