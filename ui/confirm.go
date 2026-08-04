package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lunemis/mux/tmux"
)

type confirmKillModel struct {
	sessionName string // display name
	target      string // -t target (session ID)
}

type sessionKilledMsg struct {
	name string
	err  error
}

func newConfirmKillModel(sessionName, target string) confirmKillModel {
	return confirmKillModel{sessionName: sessionName, target: target}
}

func (m confirmKillModel) Update(msg tea.Msg) (confirmKillModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			name := m.sessionName
			err := tmux.KillSession(m.target)
			return m, func() tea.Msg {
				return sessionKilledMsg{name: name, err: err}
			}
		default:
			// Any other key cancels
			return m, func() tea.Msg {
				return sessionKilledMsg{name: "", err: nil}
			}
		}
	}
	return m, nil
}

func (m confirmKillModel) View() string {
	return errorStyle.Render(
		fmt.Sprintf("Kill \"%s\"? (y/N)", m.sessionName))
}
