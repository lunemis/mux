package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefaultKeyMapContainsEveryTmuxPeekerAction(t *testing.T) {
	want := map[string]map[string][]string{
		"global": {
			"quit": {"ctrl+c"},
		},
		"list": {
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
		"create": {
			"submit": {"enter"},
			"cancel": {"esc"},
		},
		"rename": {
			"submit": {"enter"},
			"cancel": {"esc"},
		},
		"filter": {
			"apply": {"enter"},
			"clear": {"esc"},
		},
		"kill": {
			"confirm": {"y", "Y"},
			"cancel":  {"any"},
		},
		"move": {
			"up":      {"up", "k"},
			"down":    {"down", "j"},
			"confirm": {"enter"},
			"cancel":  {"esc"},
		},
	}

	got := DefaultKeyMap()
	if keys := got.Keys(contextCreate, "switch_field"); len(keys) != 0 {
		t.Errorf("create.switch_field = %#v, want removed action", keys)
	}
	for context, actions := range want {
		for action, keys := range actions {
			if actual := got.Keys(context, action); !reflect.DeepEqual(actual, keys) {
				t.Errorf("%s.%s = %#v, want %#v", context, action, actual, keys)
			}
		}
	}
}

func TestNewKeyMapAppliesPartialOverridesAndKeepsDefaults(t *testing.T) {
	got, err := NewKeyMap(map[string]map[string][]string{
		"list": {
			"up":   {"w"},
			"down": {"s", "ctrl+n"},
		},
		"create": {
			"submit": {"ctrl+s"},
		},
	})
	if err != nil {
		t.Fatalf("NewKeyMap() error = %v", err)
	}

	assertKeys(t, got, "list", "up", []string{"w"})
	assertKeys(t, got, "list", "down", []string{"s", "ctrl+n"})
	assertKeys(t, got, "create", "submit", []string{"ctrl+s"})
	assertKeys(t, got, "list", "attach", []string{"enter", "backspace"})

	if !got.Matches("list", "up", "w") {
		t.Error("custom list.up does not match w")
	}
	if got.Matches("list", "up", "k") {
		t.Error("replaced default list.up still matches k")
	}
}

func TestNewKeyMapRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]map[string][]string
		wantError string
	}{
		{
			name:      "unknown context",
			overrides: map[string]map[string][]string{"popup": {"open": {"p"}}},
			wantError: "unknown keybinding context \"popup\"",
		},
		{
			name:      "unknown action",
			overrides: map[string]map[string][]string{"list": {"explode": {"e"}}},
			wantError: "unknown keybinding action \"list.explode\"",
		},
		{
			name:      "removed create field switch",
			overrides: map[string]map[string][]string{"create": {"switch_field": {"tab"}}},
			wantError: "unknown keybinding action \"create.switch_field\"",
		},
		{
			name:      "no keys",
			overrides: map[string]map[string][]string{"list": {"up": {}}},
			wantError: "keybinding \"list.up\" requires at least one key",
		},
		{
			name:      "empty key",
			overrides: map[string]map[string][]string{"list": {"up": {""}}},
			wantError: "keybinding \"list.up\" contains an empty key",
		},
		{
			name:      "any outside kill cancel",
			overrides: map[string]map[string][]string{"list": {"up": {"any"}}},
			wantError: "key \"any\" is only valid for kill.cancel",
		},
		{
			name: "same-context conflict",
			overrides: map[string]map[string][]string{
				"list": {"up": {"w"}, "down": {"w"}},
			},
			wantError: "key \"w\" is assigned to both list.down and list.up",
		},
		{
			name: "global conflict",
			overrides: map[string]map[string][]string{
				"global": {"quit": {"enter"}},
			},
			wantError: "key \"enter\" is assigned to both global.quit and list.attach",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewKeyMap(tt.overrides)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("NewKeyMap() error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestKeyMapAnyIsAConfigurableFallback(t *testing.T) {
	defaults := DefaultKeyMap()
	if !defaults.Matches("kill", "cancel", "n") || !defaults.Matches("kill", "cancel", "esc") {
		t.Fatal("default kill.cancel should match any key")
	}

	custom, err := NewKeyMap(map[string]map[string][]string{
		"kill": {"cancel": {"n", "esc"}},
	})
	if err != nil {
		t.Fatalf("NewKeyMap() error = %v", err)
	}
	if !custom.Matches("kill", "cancel", "n") || custom.Matches("kill", "cancel", "x") {
		t.Error("custom kill.cancel should replace the any-key fallback")
	}
}

func TestKeyMapSpaceAliasMatchesBubbleTeaSpace(t *testing.T) {
	pressed := tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}.String()
	for _, configured := range []string{"space", " "} {
		keyMap, err := NewKeyMap(map[string]map[string][]string{
			"list": {"attach": {configured}},
		})
		if err != nil {
			t.Fatalf("NewKeyMap(%q) error = %v", configured, err)
		}
		if !keyMap.Matches("list", "attach", pressed) {
			t.Errorf("configured %q does not match Bubble Tea Space %q", configured, pressed)
		}
		if got := keyMap.Help("list", "attach"); got != "space" {
			t.Errorf("Help(list.attach) = %q, want space", got)
		}
	}
}

func TestNewKeyMapRejectsSpaceAliasConflict(t *testing.T) {
	_, err := NewKeyMap(map[string]map[string][]string{
		"list": {
			"attach": {"space"},
			"help":   {" "},
		},
	})
	if err == nil || !strings.Contains(err.Error(), `key "space" is assigned to both list.attach and list.help`) {
		t.Fatalf("NewKeyMap() error = %v, want visible space conflict", err)
	}
}

func TestKeyMapHelpUsesActiveBindings(t *testing.T) {
	keyMap, err := NewKeyMap(map[string]map[string][]string{
		"list": {"up": {"w", "up"}},
	})
	if err != nil {
		t.Fatalf("NewKeyMap() error = %v", err)
	}
	if got := keyMap.Help("list", "up"); got != "w/up" {
		t.Errorf("Help(list.up) = %q, want w/up", got)
	}
}

func assertKeys(t *testing.T, keyMap KeyMap, context, action string, want []string) {
	t.Helper()
	if got := keyMap.Keys(context, action); !reflect.DeepEqual(got, want) {
		t.Errorf("%s.%s = %#v, want %#v", context, action, got, want)
	}
}
