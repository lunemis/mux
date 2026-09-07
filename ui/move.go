package ui

import (
	"fmt"
	"strings"

	"github.com/aemonge/tmux-peeker/tmux"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type moveWindowModel struct {
	sourceSession string
	sourceWindow  tmux.Window
	destinations  []string
	cursor        int
	warning       string
	err           error
}

type windowMovedMsg struct {
	source      string
	destination string
	err         error
}

type moveWindowCancelledMsg struct{}

func newMoveWindowModel(source *tmux.Session, window *tmux.Window, sessions []tmux.Session) moveWindowModel {
	model := moveWindowModel{
		sourceSession: source.Name,
		sourceWindow:  *window,
	}
	for _, session := range sessions {
		if session.Name != source.Name {
			model.destinations = append(model.destinations, session.Name)
		}
	}

	if source.WindowCount <= 1 {
		model.warning = fmt.Sprintf("Moving this final window will remove session %q.", source.Name)
	}
	if len(model.destinations) == 0 {
		model.err = fmt.Errorf("no other session is available")
	}
	return model
}

func (m moveWindowModel) Update(msg tea.Msg, keyMap KeyMap) (moveWindowModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	pressed := key.String()

	switch {
	case keyMap.Matches(contextMove, "cancel", pressed):
		return m, func() tea.Msg { return moveWindowCancelledMsg{} }
	case m.err != nil:
		return m, nil
	case keyMap.Matches(contextMove, "up", pressed):
		if m.cursor > 0 {
			m.cursor--
		}
	case keyMap.Matches(contextMove, "down", pressed):
		if m.cursor < len(m.destinations)-1 {
			m.cursor++
		}
	case keyMap.Matches(contextMove, "confirm", pressed):
		source := m.sourceSession
		index := m.sourceWindow.Index
		destination := m.destinations[m.cursor]
		return m, func() tea.Msg {
			return windowMovedMsg{
				source:      source,
				destination: destination,
				err:         tmux.MoveWindow(source, index, destination),
			}
		}
	}
	return m, nil
}

func (m moveWindowModel) View(keyMap KeyMap) string {
	var b strings.Builder
	b.WriteString(inputLabelStyle.Render(fmt.Sprintf("Move %q from %q", m.sourceWindow.Name, m.sourceSession)))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(m.err.Error()))
	} else {
		if m.warning != "" {
			b.WriteString(errorStyle.Render(m.warning))
			b.WriteString("\n\n")
		}
		b.WriteString(helpStyle.Render("Destination session:"))
		b.WriteByte('\n')
		for i, destination := range m.destinations {
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(colorText)
			if i == m.cursor {
				prefix = "> "
				style = style.Bold(true).Foreground(colorCursor).Background(colorSelected)
			}
			b.WriteString(style.Render(prefix + destination))
			b.WriteByte('\n')
		}
	}

	b.WriteByte('\n')
	if m.err == nil {
		b.WriteString(helpStyle.Render(
			keyMap.Help(contextMove, "up") + "/" + keyMap.Help(contextMove, "down") + " navigate • " +
				keyMap.Help(contextMove, "confirm") + " move • " +
				keyMap.Help(contextMove, "cancel") + " cancel"))
	} else {
		b.WriteString(helpStyle.Render(keyMap.Help(contextMove, "cancel") + " close"))
	}
	return b.String()
}
