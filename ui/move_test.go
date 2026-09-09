package ui

import (
	"strings"
	"testing"

	"github.com/aemonge/tmux-peeker/tmux"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewMoveWindowModelOffersOtherSessions(t *testing.T) {
	source := &tmux.Session{Name: "source", WindowCount: 2}
	window := &tmux.Window{Index: 3, Name: "editor"}
	model := newMoveWindowModel(source, window, []tmux.Session{
		{Name: "source"},
		{Name: "alpha"},
		{Name: "beta"},
	})

	if model.sourceSession != "source" || model.sourceWindow.Index != 3 {
		t.Fatalf("source = %q:%d, want source:3", model.sourceSession, model.sourceWindow.Index)
	}
	if got := strings.Join(model.destinations, ","); got != "alpha,beta" {
		t.Errorf("destinations = %q, want alpha,beta", got)
	}
	if model.err != nil {
		t.Fatalf("newMoveWindowModel() error = %v", model.err)
	}
}

func TestNewMoveWindowModelAllowsFinalWindowWithWarning(t *testing.T) {
	source := &tmux.Session{Name: "source", WindowCount: 1}
	window := &tmux.Window{Index: 1, Name: "editor"}
	model := newMoveWindowModel(source, window, []tmux.Session{
		{Name: "source"}, {Name: "other"},
	})

	if model.err != nil {
		t.Fatalf("newMoveWindowModel() error = %v", model.err)
	}
	if !strings.Contains(model.warning, "remove session \"source\"") {
		t.Errorf("warning = %q, want source-session removal warning", model.warning)
	}
	if view := model.View(DefaultKeyMap()); !strings.Contains(view, model.warning) {
		t.Errorf("View() does not show warning %q", model.warning)
	}
}

func TestNewMoveWindowModelRejectsMissingDestination(t *testing.T) {
	source := &tmux.Session{Name: "source", WindowCount: 1}
	window := &tmux.Window{Index: 1, Name: "editor"}
	model := newMoveWindowModel(source, window, []tmux.Session{{Name: "source"}})
	if model.err == nil || !strings.Contains(model.err.Error(), "no other session") {
		t.Fatalf("error = %v, want no other session", model.err)
	}
}

func TestMoveWindowModelNavigationAndCancelUseConfiguredKeys(t *testing.T) {
	keyMap := mustKeyMap(t, map[string]map[string][]string{
		"move": {"down": {"n"}, "up": {"p"}, "cancel": {"x"}},
	})
	source := &tmux.Session{Name: "source", WindowCount: 2}
	window := &tmux.Window{Index: 1, Name: "editor"}
	model := newMoveWindowModel(source, window, []tmux.Session{
		{Name: "source"}, {Name: "alpha"}, {Name: "beta"},
	})

	model, _ = model.Update(runeKey("n"), keyMap)
	if model.cursor != 1 {
		t.Fatalf("custom down cursor = %d, want 1", model.cursor)
	}
	model, _ = model.Update(runeKey("p"), keyMap)
	if model.cursor != 0 {
		t.Fatalf("custom up cursor = %d, want 0", model.cursor)
	}
	_, cmd := model.Update(runeKey("x"), keyMap)
	if cmd == nil {
		t.Fatal("custom cancel returned no command")
	}
	if _, ok := cmd().(moveWindowCancelledMsg); !ok {
		t.Fatalf("cancel command returned %T, want moveWindowCancelledMsg", cmd())
	}
}

func TestMoveWindowModelConfirmReturnsMoveCommand(t *testing.T) {
	source := &tmux.Session{Name: "source", WindowCount: 2}
	window := &tmux.Window{Index: 1, Name: "editor"}
	model := newMoveWindowModel(source, window, []tmux.Session{
		{Name: "source"}, {Name: "destination"},
	})

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter}, DefaultKeyMap())
	if cmd == nil {
		t.Fatal("confirm returned no move command")
	}
}

func TestMoveWindowModelViewExplainsSelectionAndKeys(t *testing.T) {
	source := &tmux.Session{Name: "source", WindowCount: 2}
	window := &tmux.Window{Index: 1, Name: "editor"}
	model := newMoveWindowModel(source, window, []tmux.Session{
		{Name: "source"}, {Name: "destination"},
	})

	view := model.View(DefaultKeyMap())
	for _, want := range []string{"editor", "source", "destination", "enter", "esc"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() does not contain %q: %q", want, view)
		}
	}
}
