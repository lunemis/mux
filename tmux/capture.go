package tmux

import (
	"fmt"
	"strings"
)

// CapturePane returns the visible content of the active pane in the given session.
// Equivalent to CapturePaneTarget(sessionName).
func CapturePane(sessionName string) (string, error) {
	return CapturePaneTarget(sessionName)
}

// CapturePaneTarget captures any pane addressed by tmux's target syntax.
//
// Examples:
//   - "session"             — active pane of the active window
//   - "session:1"           — active pane of window 1
//   - "session:1.2"         — pane 2 inside window 1
func CapturePaneTarget(target string) (string, error) {
	out, err := runner.Output("tmux", "capture-pane", "-t", target, "-p", "-e")
	if err != nil {
		return "", fmt.Errorf("capture pane %s: %w", target, err)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}
