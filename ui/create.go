package ui

import (
	"github.com/aemonge/tmux-peeker/tmux"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type createModel struct {
	nameInput textinput.Model
	err       error
}

func newCreateModel() createModel {
	name := textinput.New()
	name.Placeholder = "session-name"
	name.Focus()
	name.CharLimit = 50
	name.Width = 40

	return createModel{nameInput: name}
}

type sessionCreatedMsg struct {
	name string
}

func (m createModel) Update(msg tea.Msg, keyMap KeyMap) (createModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		pressed := key.String()
		switch {
		case keyMap.Matches(contextCreate, "submit", pressed):
			name := m.nameInput.Value()
			if name == "" {
				return m, nil
			}

			err := tmux.CreateSession(name)
			if err != nil {
				m.err = err
				return m, nil
			}
			return m, func() tea.Msg {
				return sessionCreatedMsg{name: name}
			}
		}
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m createModel) View(keyMap KeyMap) string {
	s := inputLabelStyle.Render("New Session") + "\n\n"
	s += inputLabelStyle.Render("Name: ") + m.nameInput.View() + "\n\n"
	s += helpStyle.Render(keyMap.Help(contextCreate, "submit") + " create • " +
		keyMap.Help(contextCreate, "cancel") + " cancel")

	if m.err != nil {
		s += "\n" + errorStyle.Render(m.err.Error())
	}

	return s
}
