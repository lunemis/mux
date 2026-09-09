package theme

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

type themeFile struct {
	Name   string            `json:"name"`
	Colors map[string]string `json:"colors"`
}

func TestDefaultThemeContainsEveryColorRole(t *testing.T) {
	data, err := os.ReadFile("default.json")
	if err != nil {
		t.Fatal(err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("parse default theme fields: %v", err)
	}
	if len(fields) != 2 || fields["name"] == nil || fields["colors"] == nil {
		t.Errorf("default theme fields = %v, want only name and colors", fields)
	}

	var got themeFile
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse default theme: %v", err)
	}

	if got.Name != "default" {
		t.Errorf("name = %q, want default", got.Name)
	}

	for _, role := range []string{
		"primary", "accent", "success", "danger", "muted",
		"border", "separator", "surface", "selected", "cursor", "text",
	} {
		assertHexColor(t, "colors."+role, got.Colors[role])
	}
}

func TestSolarizedGruvboxThemeUsesRequestedPalette(t *testing.T) {
	data, err := os.ReadFile("solarized-gruvbox.json")
	if err != nil {
		t.Fatal(err)
	}

	var got themeFile
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse solarized gruvbox theme: %v", err)
	}

	if got.Name != "solarized-gruvbox" {
		t.Errorf("name = %q, want solarized-gruvbox", got.Name)
	}
	want := map[string]string{
		"background": "NONE",
		"text":       "#3C3836",
		"danger":     "#9D0006",
		"muted":      "#928374",
		"primary":    "#076678",
		"accent":     "#427B58",
		"success":    "#79740E",
		"separator":  "#076678",
		"surface":    "#FBF1C7",
		"cursor":     "#8F3F71",
	}
	for role, color := range want {
		if got.Colors[role] != color {
			t.Errorf("colors.%s = %q, want %s", role, got.Colors[role], color)
		}
	}
}

func TestLoadDefaultTheme(t *testing.T) {
	data, err := os.ReadFile("default.json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Load(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Name != "default" {
		t.Errorf("Name = %q, want default", got.Name)
	}
	if got.Colors.Text != "#9CA3AF" {
		t.Errorf("Colors.Text = %q, want #9CA3AF", got.Colors.Text)
	}
	if got.Colors.Separator != "#2563EB" {
		t.Errorf("Colors.Separator = %q, want #2563EB", got.Colors.Separator)
	}
	if got.Colors.Surface != "#111827" {
		t.Errorf("Colors.Surface = %q, want #111827", got.Colors.Surface)
	}
}

func TestGetLoadsEmbeddedThemeByName(t *testing.T) {
	got, err := Get("solarized-gruvbox")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Colors.Background != "NONE" {
		t.Errorf("Colors.Background = %q, want NONE", got.Colors.Background)
	}

	names := Names()
	if !slices.IsSorted(names) {
		t.Errorf("Names() = %q, want sorted names", names)
	}
	for _, name := range []string{"default", "solarized-gruvbox"} {
		if !slices.Contains(names, name) {
			t.Errorf("Names() = %q, missing %q", names, name)
		}
	}
}

func TestGetRejectsUnknownTheme(t *testing.T) {
	_, err := Get("unknown")
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("Get() error = %v, want unknown theme error", err)
	}
}

func TestLoadRejectsMissingRequiredColor(t *testing.T) {
	data, err := os.ReadFile("default.json")
	if err != nil {
		t.Fatal(err)
	}

	var raw themeFile
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw.Colors, "border")
	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(bytes.NewReader(data))
	if err == nil || !strings.Contains(err.Error(), "border") {
		t.Fatalf("Load() error = %v, want missing border error", err)
	}
}

func TestLoadRejectsMissingSurfaceColor(t *testing.T) {
	data, err := os.ReadFile("default.json")
	if err != nil {
		t.Fatal(err)
	}

	var raw themeFile
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw.Colors, "surface")
	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(bytes.NewReader(data))
	if err == nil || !strings.Contains(err.Error(), "surface") {
		t.Fatalf("Load() error = %v, want missing surface error", err)
	}
}

func TestLoadRejectsMissingSeparatorColor(t *testing.T) {
	data, err := os.ReadFile("default.json")
	if err != nil {
		t.Fatal(err)
	}

	var raw themeFile
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw.Colors, "separator")
	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(bytes.NewReader(data))
	if err == nil || !strings.Contains(err.Error(), "separator") {
		t.Fatalf("Load() error = %v, want missing separator error", err)
	}
}

func TestProductionGoFilesDoNotHardCodeHexColors(t *testing.T) {
	hexColor := regexp.MustCompile(`#[0-9A-Fa-f]{6}`)
	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if color := hexColor.Find(data); color != nil {
			t.Errorf("%s contains hard-coded color %q", path, color)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertHexColor(t *testing.T, name, value string) {
	t.Helper()
	if !regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(value) {
		t.Errorf("%s = %q, want a six-digit hex color", name, value)
	}
}
