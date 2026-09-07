package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lunemis/mux/tmux"
)

const (
	indentWindow = 2
	indentPane   = 4
)

// renderListView renders the flattened tree (sessions + expanded windows + panes).
// Items must already be flattened by the caller via flatten().
func renderListView(items []listItem, cursor int, filter string, t *treeState, width, height int) string {
	innerWidth := width - 2 // border chars
	innerHeight := height - 2

	if len(items) == 0 {
		msg := "No tmux sessions found"
		if filter != "" {
			msg = fmt.Sprintf("No match: \"%s\"", filter)
		}
		lines := make([]string, innerHeight)
		mid := innerHeight / 2
		for i := range lines {
			if i == mid {
				lines[i] = padOrTruncate(centerText(msg, innerWidth), innerWidth)
			} else {
				lines[i] = strings.Repeat(" ", innerWidth)
			}
		}
		content := strings.Join(lines, "\n")
		return drawBorder(content, width, innerHeight)
	}

	offset := 0
	if cursor >= innerHeight {
		offset = cursor - innerHeight + 1
	}

	lines := make([]string, innerHeight)
	for i := 0; i < innerHeight; i++ {
		idx := i + offset
		if idx < len(items) {
			lines[i] = formatItemRow(items[idx], idx == cursor, innerWidth, t)
		} else {
			lines[i] = strings.Repeat(" ", innerWidth)
		}
	}

	content := strings.Join(lines, "\n")
	return drawBorder(content, width, innerHeight)
}

// renderSessionList preserves the legacy session-only renderer for tests and
// callers that don't need tree expansion. It wraps each session in a listItem
// and delegates to renderListView with an empty tree state.
func renderSessionList(sessions []tmux.Session, cursor int, filter string, width, height int) string {
	items := make([]listItem, len(sessions))
	for i := range sessions {
		items[i] = listItem{kind: itemSession, session: &sessions[i]}
	}
	state := newTreeState()
	return renderListView(items, cursor, filter, &state, width, height)
}

func formatItemRow(it listItem, selected bool, width int, t *treeState) string {
	switch it.kind {
	case itemWindow:
		expanded := t.isWindowExpanded(it.session.Name, it.window.Index)
		return formatWindowRow(it.window, expanded, selected, width)
	case itemPane:
		return formatPaneRow(it.pane, selected, width)
	default:
		expanded := t.isSessionExpanded(it.session.Name)
		return formatSessionRow(*it.session, expanded, selected, width)
	}
}

func formatSessionRow(s tmux.Session, expanded, selected bool, width int) string {
	chevron := "▶"
	if expanded {
		chevron = "▼"
	}

	status := "○"
	if s.Attached {
		status = "*"
	}

	name := s.Name
	if len(name) > maxSessionNameDisplay {
		name = name[:maxSessionNameDisplay-3] + "..."
	}

	ago := timeAgo(s.Created)

	command := ""
	if s.ActiveCommand != "" {
		command = " " + s.ActiveCommand
	}

	branch := ""
	if s.GitBranch != "" {
		branch = " " + s.GitBranch
	}

	text := fmt.Sprintf("%s %s %-18s %s", chevron, status, name, ago)
	row := padOrTruncate(text+command+branch, width)

	if selected {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCursor).
			Background(colorSelected).
			Render(row)
	}

	return lipgloss.NewStyle().
		Foreground(colorText).
		Render(row)
}

func formatWindowRow(w *tmux.Window, expanded, selected bool, width int) string {
	chevron := "▶"
	if expanded {
		chevron = "▼"
	}
	marker := " "
	if w.Active {
		marker = "*"
	}

	name := w.Name
	if len(name) > maxSessionNameDisplay {
		name = name[:maxSessionNameDisplay-3] + "..."
	}

	text := fmt.Sprintf("%s%s %s %d:%s", strings.Repeat(" ", indentWindow), chevron, marker, w.Index, name)
	row := padOrTruncate(text, width)

	if selected {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCursor).
			Background(colorSelected).
			Render(row)
	}
	return lipgloss.NewStyle().
		Foreground(colorText).
		Render(row)
}

func formatPaneRow(p *tmux.Pane, selected bool, width int) string {
	marker := " "
	if p.Active {
		marker = "*"
	}
	cmd := p.Command
	if len(cmd) > maxSessionNameDisplay {
		cmd = cmd[:maxSessionNameDisplay-3] + "..."
	}

	text := fmt.Sprintf("%s%s %d %s", strings.Repeat(" ", indentPane), marker, p.Index, cmd)
	row := padOrTruncate(text, width)

	if selected {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCursor).
			Background(colorSelected).
			Render(row)
	}
	return lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(row)
}

func centerText(s string, width int) string {
	pad := (width - len(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%3ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%3dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%3dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%3dd", int(d.Hours()/24))
	}
}
