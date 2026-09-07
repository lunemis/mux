package ui

import (
	"fmt"
	"sort"
	"strings"
)

const (
	contextGlobal = "global"
	contextList   = "list"
	contextCreate = "create"
	contextRename = "rename"
	contextFilter = "filter"
	contextKill   = "kill"
	contextMove   = "move"
)

var contextOrder = []string{
	contextList,
	contextCreate,
	contextRename,
	contextFilter,
	contextKill,
	contextMove,
}

var defaultBindings = map[string]map[string][]string{
	contextGlobal: {
		"quit": {"ctrl+c"},
	},
	contextList: {
		"up":           {"up", "k"},
		"down":         {"down", "j"},
		"first":        {"g"},
		"last":         {"G"},
		"expand":       {"tab", "right", "l"},
		"collapse":     {"shift+tab", "left", "h"},
		"attach":       {"enter", "backspace"},
		"help":         {"?"},
		"create":       {"n"},
		"kill":         {"x"},
		"move_window":  {"m"},
		"rename":       {"r"},
		"filter":       {"/"},
		"clear_filter": {"esc"},
		"quit":         {"q"},
	},
	contextCreate: {
		"switch_field": {"tab", "shift+tab"},
		"submit":       {"enter"},
		"cancel":       {"esc"},
	},
	contextRename: {
		"submit": {"enter"},
		"cancel": {"esc"},
	},
	contextFilter: {
		"apply": {"enter"},
		"clear": {"esc"},
	},
	contextKill: {
		"confirm": {"y", "Y"},
		"cancel":  {"any"},
	},
	contextMove: {
		"up":      {"up", "k"},
		"down":    {"down", "j"},
		"confirm": {"enter"},
		"cancel":  {"esc"},
	},
}

// KeyMap contains the active keybindings for every tmux-peeker interaction context.
type KeyMap struct {
	bindings map[string]map[string][]string
}

// DefaultKeyMap returns tmux-peeker's built-in keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{bindings: cloneBindings(defaultBindings)}
}

// NewKeyMap applies partial user overrides to the built-in keybindings.
func NewKeyMap(overrides map[string]map[string][]string) (KeyMap, error) {
	bindings := cloneBindings(defaultBindings)

	for context, actions := range overrides {
		knownActions, ok := bindings[context]
		if !ok {
			return KeyMap{}, fmt.Errorf("unknown keybinding context %q", context)
		}
		for action, keys := range actions {
			if _, ok := knownActions[action]; !ok {
				return KeyMap{}, fmt.Errorf("unknown keybinding action %q", context+"."+action)
			}
			if len(keys) == 0 {
				return KeyMap{}, fmt.Errorf("keybinding %q requires at least one key", context+"."+action)
			}
			for _, key := range keys {
				if key == "" {
					return KeyMap{}, fmt.Errorf("keybinding %q contains an empty key", context+"."+action)
				}
				if key == "any" && (context != contextKill || action != "cancel") {
					return KeyMap{}, fmt.Errorf("key %q is only valid for kill.cancel", key)
				}
			}
			knownActions[action] = append([]string(nil), keys...)
		}
	}

	if err := validateConflicts(bindings); err != nil {
		return KeyMap{}, err
	}
	return KeyMap{bindings: bindings}, nil
}

// Keys returns the configured keys for one context and action.
func (k KeyMap) Keys(context, action string) []string {
	return append([]string(nil), k.bindings[context][action]...)
}

// Matches reports whether a Bubble Tea key string triggers an action.
func (k KeyMap) Matches(context, action, pressed string) bool {
	pressed = canonicalKey(pressed)
	for _, configured := range k.bindings[context][action] {
		if configured == "any" || canonicalKey(configured) == pressed {
			return true
		}
	}
	return false
}

// Help formats an action's active keys for display.
func (k KeyMap) Help(context, action string) string {
	keys := k.bindings[context][action]
	visible := make([]string, len(keys))
	for i, key := range keys {
		visible[i] = visibleKey(key)
	}
	return strings.Join(visible, "/")
}

func canonicalKey(key string) string {
	if key == "space" {
		return " "
	}
	return key
}

func visibleKey(key string) string {
	if canonicalKey(key) == " " {
		return "space"
	}
	return key
}

func cloneBindings(source map[string]map[string][]string) map[string]map[string][]string {
	clone := make(map[string]map[string][]string, len(source))
	for context, actions := range source {
		clone[context] = make(map[string][]string, len(actions))
		for action, keys := range actions {
			clone[context][action] = append([]string(nil), keys...)
		}
	}
	return clone
}

func validateConflicts(bindings map[string]map[string][]string) error {
	global := make(map[string]string)
	if err := conflictWithinContext(bindings, contextGlobal, global); err != nil {
		return err
	}

	for _, context := range contextOrder {
		assigned := make(map[string]string, len(global))
		for key, owner := range global {
			assigned[key] = owner
		}
		if err := conflictWithinContext(bindings, context, assigned); err != nil {
			return err
		}
	}
	return nil
}

func conflictWithinContext(bindings map[string]map[string][]string, context string, assigned map[string]string) error {
	for _, action := range sortedActions(bindings[context]) {
		owner := context + "." + action
		for _, key := range bindings[context][action] {
			if key == "any" {
				continue
			}
			identity := canonicalKey(key)
			if previous, exists := assigned[identity]; exists {
				return fmt.Errorf("key %q is assigned to both %s and %s", visibleKey(identity), previous, owner)
			}
			assigned[identity] = owner
		}
	}
	return nil
}

func sortedActions(actions map[string][]string) []string {
	names := make([]string, 0, len(actions))
	for action := range actions {
		names = append(names, action)
	}
	sort.Strings(names)
	return names
}
