package ui

import "github.com/aemonge/tmux-peeker/tmux"

// itemKind identifies whether a list row represents a session, window, or pane.
type itemKind int

const (
	itemSession itemKind = iota
	itemWindow
	itemPane
)

// listItem is one renderable row in the flattened tree view.
// For session rows, only Session is set.
// For window rows, Session and Window are set.
// For pane rows, all three are set.
type listItem struct {
	kind    itemKind
	session *tmux.Session
	window  *tmux.Window
	pane    *tmux.Pane
}

// paneCacheKey is the cache key for panes loaded from a (session, window).
type paneCacheKey struct {
	session string
	window  int
}

// treeState holds per-session expansion state and on-demand window/pane caches.
// Expansion state survives session list refreshes; window/pane caches are
// repopulated on each refresh of an expanded subtree.
type treeState struct {
	expandedSession map[string]bool
	expandedWindow  map[string]map[int]bool
	windowsCache    map[string][]tmux.Window
	panesCache      map[paneCacheKey][]tmux.Pane
}

func newTreeState() treeState {
	return treeState{
		expandedSession: make(map[string]bool),
		expandedWindow:  make(map[string]map[int]bool),
		windowsCache:    make(map[string][]tmux.Window),
		panesCache:      make(map[paneCacheKey][]tmux.Pane),
	}
}

// isSessionExpanded reports whether the session row is expanded.
func (t *treeState) isSessionExpanded(name string) bool {
	return t.expandedSession[name]
}

// isWindowExpanded reports whether the window row is expanded.
func (t *treeState) isWindowExpanded(session string, windowIdx int) bool {
	if m, ok := t.expandedWindow[session]; ok {
		return m[windowIdx]
	}
	return false
}

// setSessionExpanded toggles or sets the session expansion state.
// When collapsing a session, all its window expansions are also collapsed.
func (t *treeState) setSessionExpanded(name string, expanded bool) {
	if !expanded {
		delete(t.expandedSession, name)
		delete(t.expandedWindow, name)
		return
	}
	t.expandedSession[name] = true
}

// setWindowExpanded toggles or sets a window's expansion state.
func (t *treeState) setWindowExpanded(session string, windowIdx int, expanded bool) {
	if !expanded {
		if m, ok := t.expandedWindow[session]; ok {
			delete(m, windowIdx)
			if len(m) == 0 {
				delete(t.expandedWindow, session)
			}
		}
		return
	}
	if t.expandedWindow[session] == nil {
		t.expandedWindow[session] = make(map[int]bool)
	}
	t.expandedWindow[session][windowIdx] = true
}

// pruneCaches drops cached windows/panes/expansions for sessions no longer
// present in the given list. Called whenever the session list refreshes so
// renamed or killed sessions don't accumulate stale entries.
func (t *treeState) pruneCaches(sessions []tmux.Session) {
	live := make(map[string]struct{}, len(sessions))
	for _, s := range sessions {
		live[s.Name] = struct{}{}
	}

	for name := range t.expandedSession {
		if _, ok := live[name]; !ok {
			delete(t.expandedSession, name)
			delete(t.expandedWindow, name)
			delete(t.windowsCache, name)
		}
	}
	for key := range t.panesCache {
		if _, ok := live[key.session]; !ok {
			delete(t.panesCache, key)
		}
	}
}

// flatten builds a flattened list of rows from the given sessions and tree
// state. Expanded sessions yield their windows; expanded windows yield their
// panes. The original session slice is held by reference inside the items.
func flatten(sessions []tmux.Session, t *treeState) []listItem {
	items := make([]listItem, 0, len(sessions))
	for i := range sessions {
		s := &sessions[i]
		items = append(items, listItem{kind: itemSession, session: s})

		if !t.isSessionExpanded(s.Name) {
			continue
		}

		windows := t.windowsCache[s.Name]
		for j := range windows {
			w := &windows[j]
			items = append(items, listItem{kind: itemWindow, session: s, window: w})

			if !t.isWindowExpanded(s.Name, w.Index) {
				continue
			}

			panes := t.panesCache[paneCacheKey{session: s.Name, window: w.Index}]
			for k := range panes {
				p := &panes[k]
				items = append(items, listItem{kind: itemPane, session: s, window: w, pane: p})
			}
		}
	}
	return items
}

// canExpand reports whether the item represents a row that can be drilled into.
// Pane rows are leaves and cannot be expanded further.
func (it listItem) canExpand() bool {
	return it.kind == itemSession || it.kind == itemWindow
}

// canCollapse reports whether the item represents a row whose subtree is open
// and could be closed. Returns whether we should collapse this row, or jump
// to the parent row.
func (it listItem) canCollapse() bool {
	return it.kind == itemWindow || it.kind == itemPane
}

// previewKey identifies the (session, window, pane) tuple for capture-pane.
// window == -1 selects the session's active window; pane == -1 selects the
// window's active pane.
type previewKey struct {
	session string
	window  int
	pane    int
}

// target renders the previewKey as a tmux target string accepted by
// capture-pane (-t).
func (k previewKey) target() string {
	if k.window < 0 {
		return k.session
	}
	if k.pane < 0 {
		return formatTarget(k.session, k.window, -1)
	}
	return formatTarget(k.session, k.window, k.pane)
}

// previewKeyForItem returns the previewKey for the given list item.
func previewKeyForItem(it listItem) previewKey {
	switch it.kind {
	case itemWindow:
		return previewKey{session: it.session.Name, window: it.window.Index, pane: -1}
	case itemPane:
		return previewKey{session: it.session.Name, window: it.window.Index, pane: it.pane.Index}
	default:
		return previewKey{session: it.session.Name, window: -1, pane: -1}
	}
}
