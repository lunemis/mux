// Package theme loads the color palette used by tmux-peeker.
package theme

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
)

// Colors contains the semantic color roles used by the UI.
type Colors struct {
	Background string `json:"background,omitempty"`
	Primary    string `json:"primary"`
	Accent     string `json:"accent"`
	Success    string `json:"success"`
	Danger     string `json:"danger"`
	Muted      string `json:"muted"`
	Border     string `json:"border"`
	Separator  string `json:"separator"`
	Surface    string `json:"surface"`
	Selected   string `json:"selected"`
	Cursor     string `json:"cursor"`
	Text       string `json:"text"`
}

// Theme contains a named semantic UI palette.
type Theme struct {
	Name   string `json:"name"`
	Colors Colors `json:"colors"`
}

//go:embed *.json
var themeFiles embed.FS

var builtIns = mustLoadBuiltIns()

// Default is the built-in tmux-peeker theme used when no alternative is selected.
var Default = builtIns["default"]

// Get returns a built-in theme by name.
func Get(name string) (Theme, error) {
	value, ok := builtIns[name]
	if !ok {
		return Theme{}, fmt.Errorf("unknown theme %q (available: %s)", name, strings.Join(Names(), ", "))
	}
	return value, nil
}

// Names returns the built-in theme names in alphabetical order.
func Names() []string {
	names := make([]string, 0, len(builtIns))
	for name := range builtIns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Load decodes and validates a theme from JSON.
func Load(r io.Reader) (Theme, error) {
	var value Theme
	if err := json.NewDecoder(r).Decode(&value); err != nil {
		return Theme{}, fmt.Errorf("decode theme: %w", err)
	}
	if err := value.validate(); err != nil {
		return Theme{}, err
	}
	return value, nil
}

func (t Theme) validate() error {
	if t.Name == "" {
		return fmt.Errorf("theme name is required")
	}

	required := []struct {
		name  string
		value string
	}{
		{"primary", t.Colors.Primary},
		{"accent", t.Colors.Accent},
		{"success", t.Colors.Success},
		{"danger", t.Colors.Danger},
		{"muted", t.Colors.Muted},
		{"border", t.Colors.Border},
		{"separator", t.Colors.Separator},
		{"surface", t.Colors.Surface},
		{"selected", t.Colors.Selected},
		{"cursor", t.Colors.Cursor},
		{"text", t.Colors.Text},
	}
	for _, color := range required {
		if color.value == "" {
			return fmt.Errorf("theme color %q is required", color.name)
		}
	}
	return nil
}

func mustLoadBuiltIns() map[string]Theme {
	paths, err := fs.Glob(themeFiles, "*.json")
	if err != nil {
		panic(fmt.Sprintf("list embedded themes: %v", err))
	}

	themes := make(map[string]Theme, len(paths))
	for _, path := range paths {
		data, err := themeFiles.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("read embedded theme %s: %v", path, err))
		}
		value, err := Load(bytes.NewReader(data))
		if err != nil {
			panic(fmt.Sprintf("load embedded theme %s: %v", path, err))
		}
		if _, exists := themes[value.Name]; exists {
			panic(fmt.Sprintf("duplicate embedded theme name %q", value.Name))
		}
		themes[value.Name] = value
	}
	if _, ok := themes["default"]; !ok {
		panic("embedded default theme is missing")
	}
	return themes
}
