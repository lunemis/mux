package ui

import (
	"errors"
	"testing"

	"github.com/aemonge/tmux-peeker/tmux"
)

func TestMoveBindingOpensDestinationChooserForWindowRowOnly(t *testing.T) {
	m := modelWithSelectedWindow()
	m = updateModel(t, m, runeKey("m"))
	if m.mode != modeMoveWindow {
		t.Fatalf("move mode = %v, want modeMoveWindow", m.mode)
	}
	if m.moveWindowMod.sourceSession != "source" || m.moveWindowMod.sourceWindow.Index != 1 {
		t.Fatalf("move source = %q:%d, want source:1",
			m.moveWindowMod.sourceSession, m.moveWindowMod.sourceWindow.Index)
	}

	m = modelWithSelectedWindow()
	m.cursor = 0 // source session row
	m = updateModel(t, m, runeKey("m"))
	if m.mode != modeList {
		t.Fatalf("move on session row mode = %v, want modeList", m.mode)
	}

	m = modelWithSelectedWindow()
	m.tree.setWindowExpanded("source", 1, true)
	m.tree.panesCache[paneCacheKey{session: "source", window: 1}] = []tmux.Pane{{Index: 0}}
	m.rebuildItems()
	m.cursor = m.findItemIndex(itemPane, "source", 1, 0)
	m = updateModel(t, m, runeKey("m"))
	if m.mode != modeList {
		t.Fatalf("move on pane row mode = %v, want modeList", m.mode)
	}
}

func TestMoveBindingIsConfigurable(t *testing.T) {
	keyMap := mustKeyMap(t, map[string]map[string][]string{
		"list": {"move_window": {"v"}},
	})
	m := modelWithSelectedWindow()
	m.keyMap = keyMap

	m = updateModel(t, m, runeKey("m"))
	if m.mode != modeList {
		t.Fatalf("replaced default m mode = %v, want modeList", m.mode)
	}
	m = updateModel(t, m, runeKey("v"))
	if m.mode != modeMoveWindow {
		t.Fatalf("custom move mode = %v, want modeMoveWindow", m.mode)
	}
}

func TestWindowMovedRefreshesSourceAndDestinationTrees(t *testing.T) {
	m := modelWithSelectedWindow()
	m.mode = modeMoveWindow
	m.tree.windowsCache["destination"] = []tmux.Window{{Index: 0, Name: "shell"}}
	m.tree.expandedWindow["source"] = map[int]bool{1: true}
	m.tree.expandedWindow["destination"] = map[int]bool{0: true}

	next, cmd := m.Update(windowMovedMsg{source: "source", destination: "destination"})
	m = next.(Model)
	if m.mode != modeList {
		t.Fatalf("success mode = %v, want modeList", m.mode)
	}
	if m.focusSession != "destination" {
		t.Errorf("focusSession = %q, want destination", m.focusSession)
	}
	if _, ok := m.tree.windowsCache["source"]; ok {
		t.Error("source windows cache was not invalidated")
	}
	if _, ok := m.tree.windowsCache["destination"]; ok {
		t.Error("destination windows cache was not invalidated")
	}
	if !m.tree.isSessionExpanded("destination") {
		t.Error("destination session should be expanded after move")
	}
	if cmd == nil {
		t.Error("successful move returned no refresh command")
	}
}

func TestWindowMoveFailureStaysVisibleInChooser(t *testing.T) {
	m := modelWithSelectedWindow()
	m.mode = modeMoveWindow
	m.moveWindowMod = newMoveWindowModel(m.items[0].session, m.items[1].window, m.sessions)

	next, _ := m.Update(windowMovedMsg{
		source: "source", destination: "destination", err: errors.New("move failed"),
	})
	m = next.(Model)
	if m.mode != modeMoveWindow {
		t.Fatalf("failure mode = %v, want modeMoveWindow", m.mode)
	}
	if m.moveWindowMod.err == nil || m.moveWindowMod.err.Error() != "move failed" {
		t.Fatalf("visible move error = %v, want move failed", m.moveWindowMod.err)
	}
}

func modelWithSelectedWindow() Model {
	m := NewModel()
	m.sessions = []tmux.Session{
		{Name: "source", WindowCount: 2},
		{Name: "destination", WindowCount: 1},
	}
	m.tree.setSessionExpanded("source", true)
	m.tree.windowsCache["source"] = []tmux.Window{{Index: 1, Name: "editor"}}
	m.applyFilter()
	m.cursor = 1
	return m
}
