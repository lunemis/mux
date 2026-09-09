package ui

import (
	"testing"

	"github.com/aemonge/tmux-peeker/tmux"
)

func TestFlatten_NoExpansion(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "a"},
		{Name: "b"},
	}
	state := newTreeState()
	items := flatten(sessions, &state)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for i, it := range items {
		if it.kind != itemSession {
			t.Errorf("items[%d].kind = %d, want itemSession", i, it.kind)
		}
		if it.window != nil || it.pane != nil {
			t.Errorf("items[%d] should have nil window/pane", i)
		}
	}
}

func TestFlatten_ExpandedSession(t *testing.T) {
	sessions := []tmux.Session{{Name: "a"}, {Name: "b"}}
	state := newTreeState()
	state.setSessionExpanded("a", true)
	state.windowsCache["a"] = []tmux.Window{
		{Index: 0, Name: "editor", Active: true},
		{Index: 1, Name: "logs"},
	}

	items := flatten(sessions, &state)
	// a + 2 windows + b
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}
	if items[0].kind != itemSession || items[0].session.Name != "a" {
		t.Errorf("items[0] should be session a, got %+v", items[0])
	}
	if items[1].kind != itemWindow || items[1].window.Name != "editor" {
		t.Errorf("items[1] should be window editor, got %+v", items[1])
	}
	if items[2].kind != itemWindow || items[2].window.Name != "logs" {
		t.Errorf("items[2] should be window logs, got %+v", items[2])
	}
	if items[3].kind != itemSession || items[3].session.Name != "b" {
		t.Errorf("items[3] should be session b, got %+v", items[3])
	}
}

func TestFlatten_ExpandedWindow(t *testing.T) {
	sessions := []tmux.Session{{Name: "a"}}
	state := newTreeState()
	state.setSessionExpanded("a", true)
	state.windowsCache["a"] = []tmux.Window{
		{Index: 0, Name: "editor"},
	}
	state.setWindowExpanded("a", 0, true)
	state.panesCache[paneCacheKey{session: "a", window: 0}] = []tmux.Pane{
		{Index: 0, Command: "nvim", Active: true},
		{Index: 1, Command: "zsh"},
	}

	items := flatten(sessions, &state)
	// session a + window 0 + 2 panes
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}
	if items[2].kind != itemPane || items[2].pane.Command != "nvim" {
		t.Errorf("items[2] should be pane nvim, got %+v", items[2])
	}
	if items[3].pane.Command != "zsh" {
		t.Errorf("items[3] should be pane zsh, got %+v", items[3])
	}
}

func TestFlatten_ExpandedSessionWithoutCachedWindows(t *testing.T) {
	// Expanding a session before its windows have loaded should not crash;
	// it just yields no children until the cache populates.
	sessions := []tmux.Session{{Name: "a"}}
	state := newTreeState()
	state.setSessionExpanded("a", true)

	items := flatten(sessions, &state)
	if len(items) != 1 {
		t.Fatalf("expected 1 item (loading), got %d", len(items))
	}
}

func TestSetSessionExpanded_CollapsesWindows(t *testing.T) {
	state := newTreeState()
	state.setSessionExpanded("a", true)
	state.setWindowExpanded("a", 0, true)
	state.setWindowExpanded("a", 1, true)

	state.setSessionExpanded("a", false)

	if state.isSessionExpanded("a") {
		t.Error("session should be collapsed")
	}
	if state.isWindowExpanded("a", 0) || state.isWindowExpanded("a", 1) {
		t.Error("collapsing a session should also collapse its windows")
	}
}

func TestPruneCaches(t *testing.T) {
	state := newTreeState()
	state.setSessionExpanded("alive", true)
	state.setSessionExpanded("dead", true)
	state.windowsCache["alive"] = []tmux.Window{{Index: 0, Name: "w"}}
	state.windowsCache["dead"] = []tmux.Window{{Index: 0, Name: "w"}}
	state.setWindowExpanded("dead", 0, true)
	state.panesCache[paneCacheKey{session: "dead", window: 0}] = []tmux.Pane{{Index: 0}}
	state.panesCache[paneCacheKey{session: "alive", window: 0}] = []tmux.Pane{{Index: 0}}

	state.pruneCaches([]tmux.Session{{Name: "alive"}})

	if state.isSessionExpanded("dead") {
		t.Error("dead session expansion should be removed")
	}
	if _, ok := state.windowsCache["dead"]; ok {
		t.Error("dead session windows cache should be removed")
	}
	if _, ok := state.panesCache[paneCacheKey{session: "dead", window: 0}]; ok {
		t.Error("dead session panes cache should be removed")
	}
	if !state.isSessionExpanded("alive") {
		t.Error("alive session expansion should be preserved")
	}
	if _, ok := state.panesCache[paneCacheKey{session: "alive", window: 0}]; !ok {
		t.Error("alive session panes cache should be preserved")
	}
}

func TestSetWindowExpanded_Toggle(t *testing.T) {
	state := newTreeState()
	state.setWindowExpanded("a", 0, true)
	if !state.isWindowExpanded("a", 0) {
		t.Error("window should be expanded")
	}
	state.setWindowExpanded("a", 0, false)
	if state.isWindowExpanded("a", 0) {
		t.Error("window should be collapsed")
	}
}
