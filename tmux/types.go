// Package tmux provides functions for managing tmux sessions,
// capturing pane output, and detecting running processes.
package tmux

import "time"

// Session represents a tmux session with its metadata and state.
type Session struct {
	// ID is the tmux session id (e.g. "$5"). Unlike Name it can never contain
	// target-syntax characters (. : $ =) and survives renames, so it is the
	// value to use for every -t target.
	ID            string
	Name          string
	WindowCount   int      // total window count reported by list-sessions
	Windows       []Window // nil until enumerated via ListWindows
	Created       time.Time
	Activity      time.Time
	Attached      bool
	Directory     string
	ActiveCommand string
	PanePID       int
	GitBranch     string // current git branch, empty if not a git repo
	IsWorktree    bool   // true if Directory is a linked git worktree
}

// Window represents a single tmux window inside a session.
type Window struct {
	Index  int
	Name   string
	Active bool
	Panes  []Pane // nil until enumerated via ListPanes
}

// Pane represents a single tmux pane inside a window.
type Pane struct {
	Index   int
	Command string
	Active  bool
	Width   int
	Height  int
}
