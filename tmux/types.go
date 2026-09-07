// Package tmux provides functions for managing tmux sessions and
// capturing pane output.
package tmux

import "time"

// Session represents a tmux session with its metadata and state.
type Session struct {
	Name          string
	WindowCount   int      // total window count reported by list-sessions
	Windows       []Window // nil until enumerated via ListWindows
	Created       time.Time
	LastAttached  time.Time
	Attached      bool
	Current       bool // true when this is the invoking client's session
	Directory     string
	ActiveCommand string
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
