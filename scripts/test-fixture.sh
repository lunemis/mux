#!/usr/bin/env bash
# Sets up tmux test sessions with multiple windows and panes for issue #14
# (multi-window/pane preview) testing. Idempotent — re-running replaces
# existing fixtures.
#
# Usage:
#   scripts/test-fixture.sh up      # create fixture sessions
#   scripts/test-fixture.sh down    # remove fixture sessions
#   scripts/test-fixture.sh         # default: up

set -euo pipefail

SESSIONS=(tmux-peeker-test-multi tmux-peeker-test-empty tmux-peeker-test-deep)

teardown() {
    for s in "${SESSIONS[@]}"; do
        if tmux has-session -t "$s" 2>/dev/null; then
            tmux kill-session -t "$s"
            echo "killed: $s"
        fi
    done
}

setup() {
    teardown

    # 1) tmux-peeker-test-multi: 3 windows, mixed pane counts
    tmux new-session -d -s tmux-peeker-test-multi -n editor -c "$HOME"
    tmux send-keys -t tmux-peeker-test-multi:editor "echo 'editor: pane 0'" C-m
    tmux split-window -t tmux-peeker-test-multi:editor -h -c "$HOME"
    tmux send-keys -t tmux-peeker-test-multi:editor.1 "echo 'editor: pane 1'" C-m
    tmux split-window -t tmux-peeker-test-multi:editor.1 -v -c "$HOME"
    tmux send-keys -t tmux-peeker-test-multi:editor.2 "echo 'editor: pane 2'" C-m

    tmux new-window -t tmux-peeker-test-multi -n server -c "$HOME"
    tmux send-keys -t tmux-peeker-test-multi:server "echo 'server: pane 0'" C-m
    tmux split-window -t tmux-peeker-test-multi:server -v -c "$HOME"
    tmux send-keys -t tmux-peeker-test-multi:server.1 "echo 'server: pane 1 (try top)'" C-m

    tmux new-window -t tmux-peeker-test-multi -n logs -c "$HOME"
    tmux send-keys -t tmux-peeker-test-multi:logs "for i in 1 2 3 4 5; do echo \"log line \$i\"; done" C-m

    # 2) tmux-peeker-test-empty: just one window, one pane (baseline)
    tmux new-session -d -s tmux-peeker-test-empty -n main -c "$HOME"
    tmux send-keys -t tmux-peeker-test-empty:main "echo 'single pane session'" C-m

    # 3) tmux-peeker-test-deep: lots of windows to stress the renderer
    tmux new-session -d -s tmux-peeker-test-deep -n w0 -c "$HOME"
    for i in $(seq 1 7); do
        tmux new-window -t tmux-peeker-test-deep -n "w$i" -c "$HOME"
        tmux send-keys -t "tmux-peeker-test-deep:w$i" "echo 'window $i'" C-m
    done

    echo
    echo "fixture sessions ready:"
    tmux list-sessions | grep '^tmux-peeker-test-' || true
    echo
    echo "test:"
    echo "  ./tmux-peeker"
    echo "  → cursor on tmux-peeker-test-multi, Tab to expand windows"
    echo "  → cursor on editor, Tab to expand panes"
    echo "  → check preview updates per pane"
    echo
    echo "cleanup: scripts/test-fixture.sh down"
}

case "${1:-up}" in
    up|setup) setup ;;
    down|teardown) teardown ;;
    *) echo "usage: $0 [up|down]" >&2; exit 1 ;;
esac
