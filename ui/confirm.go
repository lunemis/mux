package ui

import (
	"fmt"

	"github.com/aemonge/tmux-peeker/tmux"
	tea "github.com/charmbracelet/bubbletea"
)

type confirmKillModel struct {
	sessionName string
}

type sessionKilledMsg struct {
	name string
	err  error
}

func newConfirmKillModel(sessionName string) confirmKillModel {
	return confirmKillModel{sessionName: sessionName}
}

func (m confirmKillModel) Update(msg tea.Msg, keyMap KeyMap) (confirmKillModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		pressed := key.String()
		switch {
		case keyMap.Matches(contextKill, "confirm", pressed):
			name := m.sessionName
			err := tmux.KillSession(name)
			return m, func() tea.Msg {
				return sessionKilledMsg{name: name, err: err}
			}
		case keyMap.Matches(contextKill, "cancel", pressed):
			return m, func() tea.Msg {
				return sessionKilledMsg{name: "", err: nil}
			}
		}
	}
	return m, nil
}

func (m confirmKillModel) View(keyMap KeyMap) string {
	return errorStyle.Render(fmt.Sprintf("Kill %q? (%s confirm / %s cancel)",
		m.sessionName,
		keyMap.Help(contextKill, "confirm"),
		keyMap.Help(contextKill, "cancel")))
}
