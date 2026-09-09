package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type filterModel struct {
	input textinput.Model
}

type filterAppliedMsg struct {
	text    string
	cleared bool
}

func newFilterModel(currentText string) filterModel {
	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = filterCharLimit
	fi.Width = filterInputWidth
	fi.SetValue(currentText)
	fi.Focus()

	return filterModel{input: fi}
}

func (m filterModel) Update(msg tea.Msg, keyMap KeyMap) (filterModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch {
		case keyMap.Matches(contextFilter, "clear", key.String()):
			return m, func() tea.Msg {
				return filterAppliedMsg{text: "", cleared: true}
			}
		case keyMap.Matches(contextFilter, "apply", key.String()):
			return m, func() tea.Msg {
				return filterAppliedMsg{text: m.input.Value(), cleared: false}
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m filterModel) View(keyMap KeyMap) string {
	return inputLabelStyle.Render(keyMap.Help(contextList, "filter")+" ") + m.input.View() +
		helpStyle.Render("  "+keyMap.Help(contextFilter, "apply")+" apply • "+
			keyMap.Help(contextFilter, "clear")+" clear")
}

func (m filterModel) LiveText() string {
	return m.input.Value()
}
