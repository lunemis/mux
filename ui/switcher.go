package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	switcherMinWidth = 36
	switcherMaxWidth = 72
	switcherMinRows  = 3
	switcherMaxRows  = 9
)

// pendingDrill records an asynchronous session/window expansion whose first
// child should receive focus once tmux returns the requested hierarchy.
type pendingDrill struct {
	childKind   itemKind
	session     string
	windowIndex int
}

type itemIdentity struct {
	valid       bool
	kind        itemKind
	session     string
	windowIndex int
	paneIndex   int
}

func (m *Model) currentIdentity() itemIdentity {
	item := m.currentItem()
	if item == nil {
		return itemIdentity{}
	}
	identity := itemIdentity{valid: true, kind: item.kind, session: item.session.Name}
	if item.window != nil {
		identity.windowIndex = item.window.Index
	}
	if item.pane != nil {
		identity.paneIndex = item.pane.Index
	}
	return identity
}

func (m *Model) restoreIdentity(identity itemIdentity) bool {
	if !identity.valid {
		return false
	}
	index := m.findItemIndex(identity.kind, identity.session, identity.windowIndex, identity.paneIndex)
	if index < 0 {
		return false
	}
	m.cursor = index
	return true
}

// selectorIndices returns the flattened-tree indices belonging to the current
// hierarchy level. The tree remains the source of truth, while the switcher
// presents only sibling sessions, windows, or panes at one time.
func (m *Model) selectInitialSwitcherTarget() bool {
	sessionIndices := make([]int, 0, len(m.items))
	currentPosition := -1
	for i, item := range m.items {
		if item.kind != itemSession {
			continue
		}
		if item.session.Current {
			currentPosition = len(sessionIndices)
		}
		sessionIndices = append(sessionIndices, i)
	}
	if len(sessionIndices) == 0 {
		return false
	}
	position := 0
	if currentPosition >= 0 && currentPosition+1 < len(sessionIndices) {
		position = currentPosition + 1
	}
	m.cursor = sessionIndices[position]
	return true
}

func (m *Model) selectorIndices() []int {
	current := m.currentItem()
	if current == nil {
		return nil
	}

	indices := make([]int, 0, len(m.items))
	for i, item := range m.items {
		switch current.kind {
		case itemSession:
			if item.kind == itemSession {
				indices = append(indices, i)
			}
		case itemWindow:
			if item.kind == itemWindow && item.session.Name == current.session.Name {
				indices = append(indices, i)
			}
		case itemPane:
			if item.kind == itemPane && item.session.Name == current.session.Name &&
				item.window.Index == current.window.Index {
				indices = append(indices, i)
			}
		}
	}
	return indices
}

func (m *Model) selectorItems() ([]listItem, int) {
	indices := m.selectorIndices()
	items := make([]listItem, 0, len(indices))
	cursor := 0
	for local, global := range indices {
		items = append(items, m.items[global])
		if global == m.cursor {
			cursor = local
		}
	}
	return items, cursor
}

func (m *Model) moveSibling(delta int) bool {
	indices := m.selectorIndices()
	for local, global := range indices {
		if global != m.cursor {
			continue
		}
		next := local + delta
		if next < 0 || next >= len(indices) {
			return false
		}
		m.cursor = indices[next]
		m.pendingDrill = nil
		return true
	}
	return false
}

func (m *Model) selectSiblingBoundary(last bool) bool {
	indices := m.selectorIndices()
	if len(indices) == 0 {
		return false
	}
	next := indices[0]
	if last {
		next = indices[len(indices)-1]
	}
	if next == m.cursor {
		return false
	}
	m.cursor = next
	m.pendingDrill = nil
	return true
}

func (m *Model) selectFirstChild(childKind itemKind, session string, windowIndex int) bool {
	for i, item := range m.items {
		if item.kind != childKind || item.session.Name != session {
			continue
		}
		if childKind == itemPane && item.window.Index != windowIndex {
			continue
		}
		m.cursor = i
		return true
	}
	return false
}

func (m *Model) finishPendingDrill(childKind itemKind, session string, windowIndex int) bool {
	pending := m.pendingDrill
	if pending == nil || pending.childKind != childKind || pending.session != session {
		return false
	}
	if childKind == itemPane && pending.windowIndex != windowIndex {
		return false
	}

	current := m.currentItem()
	parentStillSelected := current != nil && current.session.Name == session
	if childKind == itemWindow {
		parentStillSelected = parentStillSelected && current.kind == itemSession
	} else {
		parentStillSelected = parentStillSelected && current.kind == itemWindow && current.window.Index == windowIndex
	}
	m.pendingDrill = nil
	if !parentStillSelected {
		return false
	}
	return m.selectFirstChild(childKind, session, windowIndex)
}

func (m *Model) returnToSessionLevel() {
	current := m.currentItem()
	if current == nil || current.kind == itemSession {
		return
	}
	name := current.session.Name
	m.pendingDrill = nil
	m.tree.setSessionExpanded(name, false)
	m.rebuildItems()
	m.cursor = m.findItemIndex(itemSession, name, 0, 0)
}

func switcherWidth(terminalWidth int) int {
	width := terminalWidth / 2
	width = max(switcherMinWidth, min(switcherMaxWidth, width))
	return max(4, min(width, terminalWidth-4))
}

func switcherRows(itemCount, terminalHeight int) int {
	rows := max(switcherMinRows, min(switcherMaxRows, itemCount))
	return max(1, min(rows, terminalHeight-4))
}

func contextualPickerTitle(current *listItem) string {
	if current != nil {
		switch current.kind {
		case itemWindow:
			return "tmux window picker"
		case itemPane:
			return "tmux pane picker"
		}
	}
	return "tmux session picker"
}

func selectorTitle(current *listItem, count int, filter string) string {
	title := fmt.Sprintf("tmux sessions (%d)", count)
	if current != nil {
		switch current.kind {
		case itemWindow:
			title = fmt.Sprintf("%s › windows (%d)", current.session.Name, count)
		case itemPane:
			title = fmt.Sprintf("%s › %s › panes (%d)", current.session.Name, current.window.Name, count)
		}
	}
	if filter != "" {
		title += " · /" + filter
	}
	return title
}

func surfaceSpaces(count int) string {
	return lipgloss.NewStyle().
		Background(colorSurface).
		Render(strings.Repeat(" ", max(0, count)))
}

func surfacePadOrTruncate(value string, width int) string {
	value = ansi.Truncate(value, width, "")
	return value + surfaceSpaces(width-ansi.StringWidth(value))
}

func renderSwitcherSelector(m *Model) string {
	items, cursor := m.selectorItems()
	width := switcherWidth(m.width)
	itemRows := switcherRows(len(items), m.height)
	innerWidth := max(1, width-2)

	lines := make([]string, itemRows)
	if len(items) == 0 {
		message := "Loading…"
		if m.pendingDrill == nil {
			message = "No tmux sessions found"
			if m.filterText != "" {
				message = fmt.Sprintf("No match: %q", m.filterText)
			}
		}
		for i := 0; i < itemRows; i++ {
			lines[i] = surfaceSpaces(innerWidth)
		}
		lines[itemRows/2] = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSurface).
			Render(truncateAndCenter(message, innerWidth))
	} else {
		offset := 0
		if cursor >= itemRows {
			offset = cursor - itemRows + 1
		}
		for i := 0; i < itemRows; i++ {
			index := i + offset
			if index < len(items) {
				lines[i] = formatItemRow(items[index], index == cursor, innerWidth, &m.tree)
			} else {
				lines[i] = surfaceSpaces(innerWidth)
			}
		}
	}
	return drawTitledBorder(selectorTitle(m.currentItem(), len(items), m.filterText), strings.Join(lines, "\n"), width, itemRows)
}

func renderSwitcherHelp(keyMap KeyMap, terminalWidth, terminalHeight int) string {
	width := min(switcherMaxWidth, max(switcherMinWidth, terminalWidth/2))
	width = max(4, min(width, terminalWidth-4))
	innerWidth := max(1, width-2)

	entries := []struct {
		keys string
		desc string
	}{
		{keyMap.Help(contextList, "up") + " / " + keyMap.Help(contextList, "down"), "navigate"},
		{keyMap.Help(contextList, "first") + " / " + keyMap.Help(contextList, "last"), "first / last"},
		{keyMap.Help(contextList, "expand") + " / " + keyMap.Help(contextList, "collapse"), "enter / back"},
		{keyMap.Help(contextList, "attach"), "attach selected target"},
		{keyMap.Help(contextList, "create") + " / " + keyMap.Help(contextList, "rename") + " / " + keyMap.Help(contextList, "kill"), "new / rename / kill"},
		{keyMap.Help(contextList, "move_window"), "move selected window"},
		{keyMap.Help(contextList, "filter"), "filter sessions"},
		{keyMap.Help(contextList, "help") + " / esc", "close help"},
		{keyMap.Help(contextList, "quit"), "quit"},
	}

	rows := max(1, min(len(entries), terminalHeight-4))
	lines := make([]string, rows)
	for i := 0; i < rows; i++ {
		key := helpKeyStyle.Background(colorSurface).Render(entries[i].keys)
		description := helpStyle.Background(colorSurface).Render(entries[i].desc)
		padding := innerWidth - ansi.StringWidth(entries[i].keys) - ansi.StringWidth(entries[i].desc)
		if padding < 2 {
			lines[i] = surfacePadOrTruncate(key+surfaceSpaces(2)+description, innerWidth)
			continue
		}
		lines[i] = key + surfaceSpaces(padding) + description
	}
	return drawTitledBorder("help", strings.Join(lines, "\n"), width, rows)
}

func renderModal(content string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Render(content)
}
