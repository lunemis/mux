package tmux

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// #{session_id} comes first: it cannot contain the delimiter, so it stays
// intact even when a free-form field (name, path) shifts later columns.
const listFormat = "#{session_id}|#{session_name}|#{session_windows}|#{session_created}|#{session_attached}|#{pane_current_path}|#{session_activity}|#{pane_current_command}|#{pane_pid}"

// ListSessions returns all tmux sessions sorted by attached status and recent activity.
func ListSessions() ([]Session, error) {
	out, err := runner.Output("tmux", "list-sessions", "-F", listFormat)
	if err != nil {
		// tmux returns error when no server is running
		if strings.Contains(err.Error(), "exit status") {
			return nil, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	sessions := make([]Session, 0, len(lines))
	for _, line := range lines {
		s, err := parseLine(line)
		if err != nil {
			continue
		}
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Attached != sessions[j].Attached {
			return sessions[i].Attached
		}
		return sessions[i].Activity.After(sessions[j].Activity)
	})

	return sessions, nil
}

func parseLine(line string) (Session, error) {
	parts := strings.SplitN(line, "|", 9)
	if len(parts) < 9 {
		return Session{}, fmt.Errorf("unexpected format: %s", line)
	}

	windows, _ := strconv.Atoi(parts[2])
	createdUnix, _ := strconv.ParseInt(parts[3], 10, 64)
	attached, _ := strconv.Atoi(parts[4])
	activityUnix, _ := strconv.ParseInt(parts[6], 10, 64)
	panePID, _ := strconv.Atoi(parts[8])

	activeCommand := resolveCommand(panePID, parts[7])
	gitInfo := LookupGitInfo(parts[5])

	return Session{
		ID:            parts[0],
		Name:          parts[1],
		WindowCount:   windows,
		Created:       time.Unix(createdUnix, 0),
		Activity:      time.Unix(activityUnix, 0),
		Attached:      attached > 0,
		Directory:     parts[5],
		ActiveCommand: activeCommand,
		PanePID:       panePID,
		GitBranch:     gitInfo.Branch,
		IsWorktree:    gitInfo.IsWorktree,
	}, nil
}

// CreateSession creates a new detached tmux session with the given name.
func CreateSession(name string) error {
	return runner.Run("tmux", "new-session", "-d", "-s", name)
}

// CreateSessionWithDir creates a new detached tmux session starting in the given directory.
func CreateSessionWithDir(name, dir string) error {
	return runner.Run("tmux", "new-session", "-d", "-s", name, "-c", dir)
}

// KillSession destroys the tmux session with the given target.
// Pass Session.ID rather than the name: names containing target-syntax
// characters (. : $ =) cannot be addressed via -t.
func KillSession(target string) error {
	return runner.Run("tmux", "kill-session", "-t", target)
}

// RenameSession renames the session addressed by target (use Session.ID).
func RenameSession(target, newName string) error {
	return runner.Run("tmux", "rename-session", "-t", target, newName)
}
