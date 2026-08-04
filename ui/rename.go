package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lunemis/mux/tmux"
)

type renameModel struct {
	input   textinput.Model
	oldName string // display name
	target  string // -t target (session ID)
	err     error
}

type sessionRenamedMsg struct {
	oldName string
	newName string
}

func newRenameModel(oldName, target string) renameModel {
	input := textinput.New()
	input.Placeholder = oldName
	input.SetValue(oldName)
	input.Focus()
	input.CharLimit = 50
	input.Width = 40

	return renameModel{
		input:   input,
		oldName: oldName,
		target:  target,
	}
}

func (m renameModel) Update(msg tea.Msg) (renameModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			newName := m.input.Value()
			if newName == "" || newName == m.oldName {
				return m, nil
			}
			if err := tmux.RenameSession(m.target, newName); err != nil {
				m.err = err
				return m, nil
			}
			return m, func() tea.Msg {
				return sessionRenamedMsg{oldName: m.oldName, newName: newName}
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m renameModel) View() string {
	s := inputLabelStyle.Render("Rename Session") + "\n\n"
	s += inputLabelStyle.Render("Name: ") + m.input.View() + "\n\n"
	s += helpStyle.Render("enter confirm • esc cancel")

	if m.err != nil {
		s += "\n" + errorStyle.Render(m.err.Error())
	}

	return s
}
