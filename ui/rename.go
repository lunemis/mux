package ui

import (
	"github.com/aemonge/tmux-peeker/tmux"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type renameModel struct {
	input   textinput.Model
	oldName string
	err     error
}

type sessionRenamedMsg struct {
	oldName string
	newName string
}

func newRenameModel(oldName string) renameModel {
	input := textinput.New()
	input.Placeholder = oldName
	input.SetValue(oldName)
	input.Focus()
	input.CharLimit = 50
	input.Width = 40

	return renameModel{
		input:   input,
		oldName: oldName,
	}
}

func (m renameModel) Update(msg tea.Msg, keyMap KeyMap) (renameModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && keyMap.Matches(contextRename, "submit", key.String()) {
		newName := m.input.Value()
		if newName == "" || newName == m.oldName {
			return m, nil
		}
		if err := tmux.RenameSession(m.oldName, newName); err != nil {
			m.err = err
			return m, nil
		}
		return m, func() tea.Msg {
			return sessionRenamedMsg{oldName: m.oldName, newName: newName}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m renameModel) View(keyMap KeyMap) string {
	s := inputLabelStyle.Render("Rename Session") + "\n\n"
	s += inputLabelStyle.Render("Name: ") + m.input.View() + "\n\n"
	s += helpStyle.Render(keyMap.Help(contextRename, "submit") + " confirm • " +
		keyMap.Help(contextRename, "cancel") + " cancel")

	if m.err != nil {
		s += "\n" + errorStyle.Render(m.err.Error())
	}

	return s
}
