package ui

import (
	"errors"
	"testing"

	"github.com/lunemis/mux/tmux"
)

func TestSessionsLoadedClearsListWhenEmpty(t *testing.T) {
	m := NewModel()

	updated, _ := m.Update(sessionsLoadedMsg{sessions: []tmux.Session{{Name: "work"}}})
	m = updated.(Model)
	if len(m.items) != 1 {
		t.Fatalf("after loading one session: items = %d, want 1", len(m.items))
	}

	// Last session killed (or server gone): ListSessions returns (nil, nil).
	updated, _ = m.Update(sessionsLoadedMsg{sessions: nil, err: nil})
	m = updated.(Model)
	if len(m.sessions) != 0 {
		t.Errorf("sessions = %d, want 0 after empty load", len(m.sessions))
	}
	if len(m.items) != 0 {
		t.Errorf("items = %d, want 0 after empty load", len(m.items))
	}
}

func TestSessionsLoadedKeepsListOnError(t *testing.T) {
	m := NewModel()

	updated, _ := m.Update(sessionsLoadedMsg{sessions: []tmux.Session{{Name: "work"}}})
	m = updated.(Model)

	// A transient listing failure must not wipe the visible list.
	updated, _ = m.Update(sessionsLoadedMsg{sessions: nil, err: errors.New("boom")})
	m = updated.(Model)
	if len(m.items) != 1 {
		t.Errorf("items = %d, want 1 preserved after load error", len(m.items))
	}
}
