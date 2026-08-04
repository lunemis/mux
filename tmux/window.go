package tmux

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	windowListFormat = "#{window_index}|#{window_name}|#{window_active}"
	paneListFormat   = "#{pane_index}|#{pane_current_command}|#{pane_active}|#{pane_width}|#{pane_height}"
)

// ListWindows returns all windows in the session addressed by target
// (use Session.ID), sorted by index.
// Returned windows have Panes == nil; call ListPanes to populate them.
func ListWindows(target string) ([]Window, error) {
	out, err := runner.Output("tmux", "list-windows", "-t", target, "-F", windowListFormat)
	if err != nil {
		return nil, fmt.Errorf("list windows %s: %w", target, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	windows := make([]Window, 0, len(lines))
	for _, line := range lines {
		w, err := parseWindowLine(line)
		if err != nil {
			continue
		}
		windows = append(windows, w)
	}

	sort.Slice(windows, func(i, j int) bool {
		return windows[i].Index < windows[j].Index
	})

	return windows, nil
}

// ListPanes returns all panes in the given window, sorted by index.
// sessionTarget addresses the session (use Session.ID); windowIndex is the
// tmux window index (as reported by ListWindows).
func ListPanes(sessionTarget string, windowIndex int) ([]Pane, error) {
	target := fmt.Sprintf("%s:%d", sessionTarget, windowIndex)
	out, err := runner.Output("tmux", "list-panes", "-t", target, "-F", paneListFormat)
	if err != nil {
		return nil, fmt.Errorf("list panes %s: %w", target, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	panes := make([]Pane, 0, len(lines))
	for _, line := range lines {
		p, err := parsePaneLine(line)
		if err != nil {
			continue
		}
		panes = append(panes, p)
	}

	sort.Slice(panes, func(i, j int) bool {
		return panes[i].Index < panes[j].Index
	})

	return panes, nil
}

func parseWindowLine(line string) (Window, error) {
	parts := strings.SplitN(line, "|", 3)
	if len(parts) < 3 {
		return Window{}, fmt.Errorf("unexpected format: %s", line)
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return Window{}, fmt.Errorf("parse window index %q: %w", parts[0], err)
	}
	active, _ := strconv.Atoi(parts[2])

	return Window{
		Index:  index,
		Name:   parts[1],
		Active: active > 0,
	}, nil
}

func parsePaneLine(line string) (Pane, error) {
	parts := strings.SplitN(line, "|", 5)
	if len(parts) < 5 {
		return Pane{}, fmt.Errorf("unexpected format: %s", line)
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return Pane{}, fmt.Errorf("parse pane index %q: %w", parts[0], err)
	}
	active, _ := strconv.Atoi(parts[2])
	width, _ := strconv.Atoi(parts[3])
	height, _ := strconv.Atoi(parts[4])

	return Pane{
		Index:   index,
		Command: parts[1],
		Active:  active > 0,
		Width:   width,
		Height:  height,
	}, nil
}
