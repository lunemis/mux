// Package ui implements the Bubble Tea TUI for browsing and managing tmux sessions.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lunemis/mux/tmux"
)

const (
	// Timing
	refreshInterval = 500 * time.Millisecond

	// Display limits
	maxSessionNameDisplay = 18
	filterCharLimit       = 50
	filterInputWidth      = 30
)

type mode int

const (
	modeList mode = iota
	modeCreate
	modeRename
	modeFilter
	modeConfirmKill
	modeMoveWindow
)

// Model is the top-level Bubble Tea model for the session manager TUI.
type Model struct {
	keyMap         KeyMap
	sessions       []tmux.Session
	filtered       []tmux.Session
	items          []listItem // flattened tree of (sessions, windows, panes)
	tree           treeState
	cursor         int
	mode           mode
	helpVisible    bool
	pendingDrill   *pendingDrill
	width          int
	height         int
	err            error
	createModel    createModel
	renameModel    renameModel
	filterMod      filterModel
	confirmKillMod confirmKillModel
	moveWindowMod  moveWindowModel
	filterText     string
	attachTarget   previewKey // set when we want to attach after quitting (zero value = no attach)
	focusSession   string     // session name to focus cursor on after next load
	previewContent string     // cached capture-pane output
	previewKey     previewKey // (session, window, pane) the cache belongs to
	previewPrimed  bool       // prevents session refreshes from duplicating the first capture
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type sessionsLoadedMsg struct {
	sessions []tmux.Session
	err      error
}

func loadSessions() tea.Msg {
	sessions, err := tmux.ListSessions()
	return sessionsLoadedMsg{sessions: sessions, err: err}
}

type previewLoadedMsg struct {
	key     previewKey
	content string
}

type windowsLoadedMsg struct {
	sessionName string
	windows     []tmux.Window
}

type panesLoadedMsg struct {
	sessionName string
	windowIndex int
	panes       []tmux.Pane
}

func loadWindows(sessionName string) tea.Cmd {
	return func() tea.Msg {
		windows, _ := tmux.ListWindows(sessionName)
		return windowsLoadedMsg{sessionName: sessionName, windows: windows}
	}
}

func loadPanes(sessionName string, windowIndex int) tea.Cmd {
	return func() tea.Msg {
		panes, _ := tmux.ListPanes(sessionName, windowIndex)
		return panesLoadedMsg{sessionName: sessionName, windowIndex: windowIndex, panes: panes}
	}
}

func refreshPreview(key previewKey) tea.Cmd {
	return func() tea.Msg {
		content, err := tmux.CapturePaneTarget(key.target())
		if err != nil {
			content = "Error: " + err.Error()
		}
		return previewLoadedMsg{key: key, content: content}
	}
}

// NewModel returns a new Model with mux's default keybindings.
func NewModel() Model {
	return NewModelWithKeyMap(DefaultKeyMap())
}

// NewModelWithKeyMap returns a new Model with the supplied keybindings.
func NewModelWithKeyMap(keyMap KeyMap) Model {
	return Model{keyMap: keyMap, tree: newTreeState()}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadSessions, tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.keyMap.Matches(contextGlobal, "quit", key.String()) {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{loadSessions, tick()}
		if it := m.currentItem(); it != nil {
			cmds = append(cmds, refreshPreview(previewKeyForItem(*it)))
		}
		// Refresh windows/panes for expanded subtrees
		for name := range m.tree.expandedSession {
			cmds = append(cmds, loadWindows(name))
		}
		for sessionName, windows := range m.tree.expandedWindow {
			for windowIdx := range windows {
				cmds = append(cmds, loadPanes(sessionName, windowIdx))
			}
		}
		return m, tea.Batch(cmds...)

	case sessionsLoadedMsg:
		m.err = msg.err
		if msg.sessions != nil {
			selected := m.currentIdentity()
			m.sessions = msg.sessions
			m.tree.pruneCaches(m.sessions)
			m.applyFilter()
			if m.focusSession != "" {
				for i, it := range m.items {
					if it.kind == itemSession && it.session.Name == m.focusSession {
						m.cursor = i
						break
					}
				}
				m.focusSession = ""
			} else if !m.restoreIdentity(selected) {
				m.selectInitialSwitcherTarget()
			}
			if !m.previewPrimed {
				if cmd := m.refreshCurrentPreview(); cmd != nil {
					m.previewPrimed = true
					return m, cmd
				}
			}
		}
		return m, nil

	case windowsLoadedMsg:
		selected := m.currentIdentity()
		m.tree.windowsCache[msg.sessionName] = msg.windows
		m.rebuildItems()
		m.restoreIdentity(selected)
		if m.finishPendingDrill(itemWindow, msg.sessionName, 0) {
			return m, m.refreshCurrentPreview()
		}
		return m, nil

	case panesLoadedMsg:
		selected := m.currentIdentity()
		m.tree.panesCache[paneCacheKey{session: msg.sessionName, window: msg.windowIndex}] = msg.panes
		m.rebuildItems()
		m.restoreIdentity(selected)
		if m.finishPendingDrill(itemPane, msg.sessionName, msg.windowIndex) {
			return m, m.refreshCurrentPreview()
		}
		return m, nil

	case previewLoadedMsg:
		m.previewKey = msg.key
		m.previewContent = msg.content
		return m, nil

	case sessionCreatedMsg:
		m.mode = modeList
		m.focusSession = msg.name
		return m, loadSessions

	case sessionRenamedMsg:
		m.mode = modeList
		return m, loadSessions

	case filterAppliedMsg:
		m.mode = modeList
		m.filterText = msg.text
		m.applyFilter()
		return m, nil

	case sessionKilledMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.mode = modeList
		if msg.name != "" {
			return m, loadSessions
		}
		return m, nil

	case moveWindowCancelledMsg:
		m.mode = modeList
		return m, nil

	case windowMovedMsg:
		if msg.err != nil {
			m.moveWindowMod.err = msg.err
			return m, nil
		}
		m.mode = modeList
		m.focusSession = msg.destination
		for _, sessionName := range []string{msg.source, msg.destination} {
			delete(m.tree.windowsCache, sessionName)
			delete(m.tree.expandedWindow, sessionName)
			for key := range m.tree.panesCache {
				if key.session == sessionName {
					delete(m.tree.panesCache, key)
				}
			}
		}
		m.tree.setSessionExpanded(msg.destination, true)
		return m, tea.Batch(
			loadSessions,
			loadWindows(msg.source),
			loadWindows(msg.destination),
		)
	}

	switch m.mode {
	case modeCreate:
		return m.updateCreate(msg)
	case modeRename:
		return m.updateRename(msg)
	case modeFilter:
		return m.updateFilter(msg)
	case modeConfirmKill:
		return m.updateConfirmKill(msg)
	case modeMoveWindow:
		return m.updateMoveWindow(msg)
	default:
		return m.updateList(msg)
	}
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	pressed := key.String()

	if m.helpVisible {
		if pressed == "esc" || m.keyMap.Matches(contextList, "help", pressed) {
			m.helpVisible = false
		}
		return m, nil
	}

	switch {
	case m.keyMap.Matches(contextList, "quit", pressed):
		return m, tea.Quit
	case m.keyMap.Matches(contextList, "help", pressed):
		m.helpVisible = true
	case m.keyMap.Matches(contextList, "up", pressed):
		if m.moveSibling(-1) {
			return m, m.refreshCurrentPreview()
		}
	case m.keyMap.Matches(contextList, "down", pressed):
		if m.moveSibling(1) {
			return m, m.refreshCurrentPreview()
		}
	case m.keyMap.Matches(contextList, "first", pressed):
		if m.selectSiblingBoundary(false) {
			return m, m.refreshCurrentPreview()
		}
	case m.keyMap.Matches(contextList, "last", pressed):
		if m.selectSiblingBoundary(true) {
			return m, m.refreshCurrentPreview()
		}
	case m.keyMap.Matches(contextList, "expand", pressed):
		return m.expandCurrent()
	case m.keyMap.Matches(contextList, "collapse", pressed):
		return m.collapseCurrent()
	case m.keyMap.Matches(contextList, "attach", pressed):
		if it := m.currentItem(); it != nil {
			m.attachTarget = previewKeyForItem(*it)
			return m, tea.Quit
		}
	case m.keyMap.Matches(contextList, "create", pressed):
		m.mode = modeCreate
		m.createModel = newCreateModel()
		return m, m.createModel.nameInput.Focus()
	case m.keyMap.Matches(contextList, "kill", pressed):
		if it := m.currentItem(); it != nil && it.kind == itemSession {
			m.mode = modeConfirmKill
			m.confirmKillMod = newConfirmKillModel(it.session.Name)
		}
	case m.keyMap.Matches(contextList, "move_window", pressed):
		if it := m.currentItem(); it != nil && it.kind == itemWindow {
			m.mode = modeMoveWindow
			m.moveWindowMod = newMoveWindowModel(it.session, it.window, m.sessions)
		}
	case m.keyMap.Matches(contextList, "rename", pressed):
		if it := m.currentItem(); it != nil && it.kind == itemSession {
			m.mode = modeRename
			m.renameModel = newRenameModel(it.session.Name)
			return m, m.renameModel.input.Focus()
		}
	case m.keyMap.Matches(contextList, "filter", pressed):
		m.returnToSessionLevel()
		m.mode = modeFilter
		m.filterMod = newFilterModel(m.filterText)
		return m, nil
	case m.keyMap.Matches(contextList, "clear_filter", pressed):
		if m.filterText != "" {
			m.filterText = ""
			m.applyFilter()
		}
	}
	return m, nil
}

// expandCurrent drills into the selected session or window. Cached children
// are selected immediately; otherwise focus moves when the async load returns.
func (m Model) expandCurrent() (tea.Model, tea.Cmd) {
	it := m.currentItem()
	if it == nil || !it.canExpand() {
		return m, nil
	}
	switch it.kind {
	case itemSession:
		name := it.session.Name
		m.tree.setSessionExpanded(name, true)
		m.rebuildItems()
		if m.selectFirstChild(itemWindow, name, 0) {
			return m, m.refreshCurrentPreview()
		}
		m.pendingDrill = &pendingDrill{childKind: itemWindow, session: name}
		return m, loadWindows(name)
	case itemWindow:
		name, index := it.session.Name, it.window.Index
		m.tree.setWindowExpanded(name, index, true)
		m.rebuildItems()
		if m.selectFirstChild(itemPane, name, index) {
			return m, m.refreshCurrentPreview()
		}
		m.pendingDrill = &pendingDrill{childKind: itemPane, session: name, windowIndex: index}
		return m, loadPanes(name, index)
	}
	return m, nil
}

// collapseCurrent returns from panes to their window or from windows to their
// session. At session level it only cancels an in-flight drill.
func (m Model) collapseCurrent() (tea.Model, tea.Cmd) {
	it := m.currentItem()
	if it == nil {
		return m, nil
	}
	m.pendingDrill = nil
	switch it.kind {
	case itemSession:
		m.tree.setSessionExpanded(it.session.Name, false)
	case itemWindow:
		name := it.session.Name
		m.tree.setSessionExpanded(name, false)
		m.rebuildItems()
		m.cursor = m.findItemIndex(itemSession, name, 0, 0)
		return m, m.refreshCurrentPreview()
	case itemPane:
		name, index := it.session.Name, it.window.Index
		m.tree.setWindowExpanded(name, index, false)
		m.rebuildItems()
		m.cursor = m.findItemIndex(itemWindow, name, index, 0)
		return m, m.refreshCurrentPreview()
	}
	m.rebuildItems()
	return m, m.refreshCurrentPreview()
}

// refreshCurrentPreview returns a tea.Cmd to capture the pane targeted by the
// current cursor position. Returns nil when there is no current item.
func (m *Model) refreshCurrentPreview() tea.Cmd {
	if it := m.currentItem(); it != nil {
		return refreshPreview(previewKeyForItem(*it))
	}
	return nil
}

// findItemIndex returns the index of the matching listItem, or -1 if not found.
func (m *Model) findItemIndex(kind itemKind, sessionName string, windowIdx, paneIdx int) int {
	for i, it := range m.items {
		if it.kind != kind || it.session.Name != sessionName {
			continue
		}
		switch kind {
		case itemSession:
			return i
		case itemWindow:
			if it.window.Index == windowIdx {
				return i
			}
		case itemPane:
			if it.window.Index == windowIdx && it.pane.Index == paneIdx {
				return i
			}
		}
	}
	return -1
}

func (m Model) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.keyMap.Matches(contextCreate, "cancel", key.String()) {
		m.mode = modeList
		return m, nil
	}
	var cmd tea.Cmd
	m.createModel, cmd = m.createModel.Update(msg, m.keyMap)
	return m, cmd
}

func (m Model) updateRename(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.keyMap.Matches(contextRename, "cancel", key.String()) {
		m.mode = modeList
		return m, nil
	}
	var cmd tea.Cmd
	m.renameModel, cmd = m.renameModel.Update(msg, m.keyMap)
	return m, cmd
}

func (m Model) updateFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.filterMod, cmd = m.filterMod.Update(msg, m.keyMap)
	// Live filter as you type
	m.filterText = m.filterMod.LiveText()
	m.applyFilter()
	return m, cmd
}

func (m Model) updateConfirmKill(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirmKillMod, cmd = m.confirmKillMod.Update(msg, m.keyMap)
	return m, cmd
}

func (m Model) updateMoveWindow(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.moveWindowMod, cmd = m.moveWindowMod.Update(msg, m.keyMap)
	return m, cmd
}

func (m *Model) currentItem() *listItem {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return &m.items[m.cursor]
	}
	return nil
}

// currentSession returns the parent session of the current row (the row itself
// for session rows). Returns nil if no row is selected.
func (m *Model) currentSession() *tmux.Session {
	if it := m.currentItem(); it != nil {
		return it.session
	}
	return nil
}

func (m *Model) currentSessionName() string {
	if s := m.currentSession(); s != nil {
		return s.Name
	}
	return ""
}

// rebuildItems recomputes the flattened tree view from the filtered session
// list and current expansion state. Call after sessions, filter, or expansion
// state changes.
func (m *Model) rebuildItems() {
	m.items = flatten(m.filtered, &m.tree)
	if m.cursor >= len(m.items) {
		m.cursor = max(0, len(m.items)-1)
	}
}

func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.filtered = m.sessions
	} else {
		lower := strings.ToLower(m.filterText)
		m.filtered = nil
		for _, s := range m.sessions {
			if strings.Contains(strings.ToLower(s.Name), lower) ||
				strings.Contains(strings.ToLower(s.Directory), lower) {
				m.filtered = append(m.filtered, s)
			}
		}
	}
	m.rebuildItems()
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var view string
	switch m.mode {
	case modeCreate:
		view = m.viewWithOverlay(m.createModel.View(m.keyMap))
	case modeRename:
		view = m.viewWithOverlay(m.renameModel.View(m.keyMap))
	case modeFilter:
		view = m.viewWithOverlay(m.filterMod.View(m.keyMap))
	case modeConfirmKill:
		view = m.viewWithOverlay(m.confirmKillMod.View(m.keyMap))
	case modeMoveWindow:
		view = m.viewWithOverlay(m.moveWindowMod.View(m.keyMap))
	default:
		view = m.viewMain()
	}
	if applyBackground {
		return lipgloss.NewStyle().Background(colorBackground).Render(view)
	}
	return view
}

func (m Model) previewBackground() string {
	if m.height < minimumSwitcherHeight || m.width < 20 {
		return fixedBox(errorStyle.Render("Terminal too small for mux"), m.width, m.height)
	}

	currentItem := m.currentItem()
	cachedContent := ""
	if currentItem != nil && m.previewKey == previewKeyForItem(*currentItem) {
		cachedContent = m.previewContent
	}
	title := truncateAndCenter(titleStyle.Render(contextualPickerTitle(currentItem)), m.width)
	preview := renderPreview(cachedContent, m.width, m.height-1)
	return title + "\n" + preview
}

func (m Model) viewMain() string {
	background := m.previewBackground()
	if m.height < minimumSwitcherHeight || m.width < 20 {
		return background
	}
	overlay := renderSwitcherSelector(&m)
	if m.helpVisible {
		overlay = renderSwitcherHelp(m.keyMap, m.width, m.height)
	}
	return overlayCentered(background, overlay, m.width, m.height)
}

func (m Model) viewWithOverlay(content string) string {
	background := m.previewBackground()
	if m.height < minimumSwitcherHeight || m.width < 20 {
		return background
	}
	return overlayCentered(background, renderModal(content), m.width, m.height)
}

// AttachName returns the session name to attach to (if any) after the TUI
// exits. Returns empty when no attach was requested.
func (m Model) AttachName() string {
	return m.attachTarget.session
}

// AttachWindowIndex returns the window index selected for attachment, or -1
// if the user selected a session row.
func (m Model) AttachWindowIndex() int {
	return m.attachTarget.window
}

// AttachPaneIndex returns the pane index selected for attachment, or -1 if
// the user did not drill down to a pane row.
func (m Model) AttachPaneIndex() int {
	return m.attachTarget.pane
}

// AttachToSession switches to the target session, optionally focusing a
// specific window and pane first. Pass windowIdx == -1 to keep the active
// window; pass paneIdx == -1 to keep the active pane within that window.
//
// If already inside tmux, uses switch-client. Otherwise, uses attach-session.
func AttachToSession(name string, windowIdx, paneIdx int) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	// Focus the requested window/pane *before* attaching, since attach-session
	// replaces our process and we can't run anything afterwards.
	if windowIdx >= 0 {
		windowTarget := fmt.Sprintf("%s:%d", name, windowIdx)
		if err := exec.Command(tmuxPath, "select-window", "-t", windowTarget).Run(); err != nil {
			return fmt.Errorf("select-window %s: %w", windowTarget, err)
		}
		if paneIdx >= 0 {
			paneTarget := fmt.Sprintf("%s.%d", windowTarget, paneIdx)
			if err := exec.Command(tmuxPath, "select-pane", "-t", paneTarget).Run(); err != nil {
				return fmt.Errorf("select-pane %s: %w", paneTarget, err)
			}
		}
	}

	if os.Getenv("TMUX") != "" {
		return exec.Command(tmuxPath, "switch-client", "-t", name).Run()
	}
	return syscall.Exec(tmuxPath, []string{"tmux", "attach-session", "-t", name}, os.Environ())
}
