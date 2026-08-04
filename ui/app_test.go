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

// 세션 이름에 . : $ = 가 들어가면 -t 타깃 문법과 충돌해 kill/capture가
// 실패하므로, 타깃은 항상 세션 ID($N)로 만들어야 한다.
func TestPreviewKeyForItemUsesSessionID(t *testing.T) {
	sess := &tmux.Session{Name: "work.dev", ID: "$5"}
	win := &tmux.Window{Index: 1}
	pane := &tmux.Pane{Index: 2}

	tests := []struct {
		item listItem
		want string
	}{
		{listItem{kind: itemSession, session: sess}, "$5"},
		{listItem{kind: itemWindow, session: sess, window: win}, "$5:1"},
		{listItem{kind: itemPane, session: sess, window: win, pane: pane}, "$5:1.2"},
	}
	for _, tt := range tests {
		got := previewKeyForItem(tt.item).target()
		if got != tt.want {
			t.Errorf("previewKeyForItem(%v).target() = %q, want %q", tt.item.kind, got, tt.want)
		}
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
