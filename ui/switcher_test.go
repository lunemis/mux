package ui

import (
	"strings"
	"testing"

	"github.com/aemonge/tmux-peeker/tmux"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestSwitcherInitialSelectionStartsOnPreviousSession(t *testing.T) {
	m := NewModel()
	next, _ := m.Update(sessionsLoadedMsg{sessions: []tmux.Session{
		{Name: "current", Current: true},
		{Name: "previous"},
		{Name: "older"},
	}})
	m = next.(Model)

	if got := []string{m.items[0].session.Name, m.items[1].session.Name, m.items[2].session.Name}; got[0] != "current" || got[1] != "previous" || got[2] != "older" {
		t.Fatalf("display order = %q, want [current previous older]", got)
	}
	if item := m.currentItem(); item == nil || item.session.Name != "previous" {
		t.Fatalf("initial selection = %#v, want previous session", item)
	}
}

func TestSwitcherInitialLoadRequestsPreviewOnce(t *testing.T) {
	m := NewModel()

	next, cmd := m.Update(sessionsLoadedMsg{sessions: []tmux.Session{}})
	m = next.(Model)
	if cmd != nil {
		t.Fatal("empty initial session load should not request a preview")
	}

	sessions := []tmux.Session{{Name: "current", Current: true}, {Name: "previous"}}
	next, cmd = m.Update(sessionsLoadedMsg{sessions: sessions})
	m = next.(Model)
	if cmd == nil {
		t.Fatal("first selectable session load should request the selected session preview")
	}

	_, cmd = m.Update(sessionsLoadedMsg{sessions: sessions})
	if cmd != nil {
		t.Fatal("subsequent session refresh should not request a duplicate preview")
	}
}

func TestSwitcherInitialSelectionHandlesSingleAndOutsideTmux(t *testing.T) {
	tests := []struct {
		name     string
		sessions []tmux.Session
		want     string
	}{
		{name: "single current", sessions: []tmux.Session{{Name: "only", Current: true}}, want: "only"},
		{name: "outside tmux", sessions: []tmux.Session{{Name: "newest"}, {Name: "older"}}, want: "newest"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := NewModel()
			next, _ := m.Update(sessionsLoadedMsg{sessions: test.sessions})
			m = next.(Model)
			if item := m.currentItem(); item == nil || item.session.Name != test.want {
				t.Fatalf("initial selection = %#v, want %q", item, test.want)
			}
		})
	}
}

func TestSwitcherDrillsThroughHierarchyAndReturnsToParents(t *testing.T) {
	m := NewModel()
	m.sessions = []tmux.Session{{Name: "work"}, {Name: "other"}}
	m.applyFilter()
	m.tree.windowsCache["work"] = []tmux.Window{
		{Index: 1, Name: "editor", Active: true},
		{Index: 2, Name: "shell"},
	}

	next, _ := m.expandCurrent()
	m = next.(Model)
	if item := m.currentItem(); item == nil || item.kind != itemWindow || item.window.Index != 1 {
		t.Fatalf("session drill selected %#v, want first window", item)
	}
	items, cursor := m.selectorItems()
	if len(items) != 2 || cursor != 0 || items[1].kind != itemWindow {
		t.Fatalf("window selector = %#v cursor=%d, want two sibling windows", items, cursor)
	}

	m.tree.panesCache[paneCacheKey{session: "work", window: 1}] = []tmux.Pane{
		{Index: 0, Command: "pi", Active: true},
		{Index: 1, Command: "shell"},
	}
	next, _ = m.expandCurrent()
	m = next.(Model)
	if item := m.currentItem(); item == nil || item.kind != itemPane || item.pane.Index != 0 {
		t.Fatalf("window drill selected %#v, want first pane", item)
	}

	next, _ = m.collapseCurrent()
	m = next.(Model)
	if item := m.currentItem(); item == nil || item.kind != itemWindow || item.window.Index != 1 {
		t.Fatalf("pane back selected %#v, want parent window", item)
	}
	next, _ = m.collapseCurrent()
	m = next.(Model)
	if item := m.currentItem(); item == nil || item.kind != itemSession || item.session.Name != "work" {
		t.Fatalf("window back selected %#v, want parent session", item)
	}
}

func TestSwitcherCompletesAsynchronousDrill(t *testing.T) {
	m := NewModel()
	m.sessions = []tmux.Session{{Name: "work"}}
	m.applyFilter()

	next, cmd := m.expandCurrent()
	m = next.(Model)
	if cmd == nil || m.pendingDrill == nil {
		t.Fatal("uncached session drill should load windows and remember pending focus")
	}

	next, _ = m.Update(windowsLoadedMsg{
		sessionName: "work",
		windows:     []tmux.Window{{Index: 3, Name: "editor"}},
	})
	m = next.(Model)
	if m.pendingDrill != nil {
		t.Fatal("completed window load should clear pending drill")
	}
	if item := m.currentItem(); item == nil || item.kind != itemWindow || item.window.Index != 3 {
		t.Fatalf("loaded drill selected %#v, want returned window", item)
	}
}

func TestSwitcherNavigationCancelsStaleAsynchronousDrill(t *testing.T) {
	m := NewModel()
	m.sessions = []tmux.Session{{Name: "work"}, {Name: "other"}}
	m.applyFilter()

	next, _ := m.expandCurrent()
	m = next.(Model)
	m = updateModel(t, m, runeKey("j"))
	if m.pendingDrill != nil {
		t.Fatal("moving to another session should cancel pending drill focus")
	}

	next, _ = m.Update(windowsLoadedMsg{
		sessionName: "work",
		windows:     []tmux.Window{{Index: 3, Name: "editor"}},
	})
	m = next.(Model)
	if item := m.currentItem(); item == nil || item.kind != itemSession || item.session.Name != "other" {
		t.Fatalf("stale window response stole focus: %#v", item)
	}
}

func TestSwitcherNavigationStaysWithinCurrentLevel(t *testing.T) {
	m := NewModel()
	m.sessions = []tmux.Session{{Name: "work"}, {Name: "other"}}
	m.applyFilter()
	m.tree.windowsCache["work"] = []tmux.Window{
		{Index: 1, Name: "editor"},
		{Index: 2, Name: "shell"},
	}
	next, _ := m.expandCurrent()
	m = next.(Model)

	m = updateModel(t, m, runeKey("j"))
	if item := m.currentItem(); item == nil || item.kind != itemWindow || item.window.Index != 2 {
		t.Fatalf("down selected %#v, want sibling window", item)
	}
	m = updateModel(t, m, runeKey("j"))
	if item := m.currentItem(); item == nil || item.kind != itemWindow || item.window.Index != 2 {
		t.Fatalf("down crossed hierarchy boundary: %#v", item)
	}
}

func TestSwitcherAcceptKeysAttachSelectedPane(t *testing.T) {
	m := NewModel()
	m.sessions = []tmux.Session{{Name: "work"}}
	m.applyFilter()
	m.tree.windowsCache["work"] = []tmux.Window{{Index: 1, Name: "editor"}}
	next, _ := m.expandCurrent()
	m = next.(Model)
	m.tree.panesCache[paneCacheKey{session: "work", window: 1}] = []tmux.Pane{{Index: 2, Command: "pi"}}
	next, _ = m.expandCurrent()
	m = next.(Model)

	for name, key := range map[string]tea.KeyMsg{
		"enter":     {Type: tea.KeyEnter},
		"backspace": {Type: tea.KeyBackspace},
	} {
		t.Run(name, func(t *testing.T) {
			next, cmd := m.Update(key)
			got := next.(Model)
			if cmd == nil {
				t.Fatal("attaching a pane should quit the switcher")
			}
			if got.attachTarget != (previewKey{session: "work", window: 1, pane: 2}) {
				t.Fatalf("attach target = %#v, want exact selected pane", got.attachTarget)
			}
		})
	}
}

func TestConfiguredSpaceAttachesSelectedSession(t *testing.T) {
	keyMap, err := NewKeyMap(map[string]map[string][]string{
		"list": {"attach": {"enter", "space"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel()
	m.keyMap = keyMap
	m.sessions = []tmux.Session{{Name: "work"}}
	m.applyFilter()

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	got := next.(Model)
	if cmd == nil {
		t.Fatal("configured Space should quit the switcher")
	}
	if got.attachTarget.session != "work" {
		t.Fatalf("attach target = %#v, want work session", got.attachTarget)
	}
}

func TestSwitcherPreservesChildSelectionAcrossSessionRefresh(t *testing.T) {
	m := NewModel()
	m.sessions = []tmux.Session{{Name: "work"}, {Name: "other"}}
	m.applyFilter()
	m.tree.windowsCache["work"] = []tmux.Window{{Index: 4, Name: "editor"}}
	next, _ := m.expandCurrent()
	m = next.(Model)

	next, _ = m.Update(sessionsLoadedMsg{sessions: []tmux.Session{{Name: "other"}, {Name: "work"}}})
	m = next.(Model)
	if item := m.currentItem(); item == nil || item.kind != itemWindow || item.session.Name != "work" || item.window.Index != 4 {
		t.Fatalf("session refresh selected %#v, want original child target", item)
	}
}

func TestContextualPickerTitleTracksHierarchy(t *testing.T) {
	session := tmux.Session{Name: "work"}
	window := tmux.Window{Index: 1, Name: "editor"}
	pane := tmux.Pane{Index: 0, Command: "pi"}

	tests := []struct {
		name string
		item *listItem
		want string
	}{
		{name: "empty", want: "tmux session picker"},
		{name: "session", item: &listItem{kind: itemSession, session: &session}, want: "tmux session picker"},
		{name: "window", item: &listItem{kind: itemWindow, session: &session, window: &window}, want: "tmux window picker"},
		{name: "pane", item: &listItem{kind: itemPane, session: &session, window: &window, pane: &pane}, want: "tmux pane picker"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := contextualPickerTitle(test.item); got != test.want {
				t.Errorf("contextualPickerTitle() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSwitcherSelectorTitleTracksHierarchy(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.sessions = []tmux.Session{{Name: "work"}}
	m.applyFilter()
	m.tree.windowsCache["work"] = []tmux.Window{{Index: 1, Name: "editor"}}
	next, _ := m.expandCurrent()
	m = next.(Model)

	windowSelector := ansi.Strip(renderSwitcherSelector(&m))
	if !strings.Contains(windowSelector, "work › windows") {
		t.Fatalf("window selector title missing context: %q", windowSelector)
	}

	m.tree.panesCache[paneCacheKey{session: "work", window: 1}] = []tmux.Pane{{Index: 0, Command: "pi"}}
	next, _ = m.expandCurrent()
	m = next.(Model)
	paneSelector := ansi.Strip(renderSwitcherSelector(&m))
	if !strings.Contains(paneSelector, "work › editor › panes") {
		t.Fatalf("pane selector title missing context: %q", paneSelector)
	}
}
