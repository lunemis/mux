package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathUsesXDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	want := filepath.Join(xdg, "tmux-peeker", "config.json")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	want := filepath.Join(home, ".config", "tmux-peeker", "config.json")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestLoadReadsTheme(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path := filepath.Join(xdg, "tmux-peeker", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"solarized-gruvbox"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Theme != "solarized-gruvbox" {
		t.Errorf("Theme = %q, want solarized-gruvbox", got.Theme)
	}
}

func TestLoadReadsKeybindings(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path := filepath.Join(xdg, "tmux-peeker", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := `{
		"theme": "solarized-gruvbox",
		"keybindings": {
			"list": {"up": ["w", "up"], "down": ["s", "down"]},
			"create": {"submit": ["ctrl+s"]}
		}
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Theme != "solarized-gruvbox" {
		t.Errorf("Theme = %q, want solarized-gruvbox", got.Theme)
	}
	if diff := strings.Join(got.Keybindings["list"]["up"], ","); diff != "w,up" {
		t.Errorf("list.up = %q, want w,up", diff)
	}
	if diff := strings.Join(got.Keybindings["create"]["submit"], ","); diff != "ctrl+s" {
		t.Errorf("create.submit = %q, want ctrl+s", diff)
	}
}

func TestLoadIgnoresUpstreamConfig(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	upstream := filepath.Join(xdg, "mux", "config.json")
	if err := os.MkdirAll(filepath.Dir(upstream), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(upstream, []byte(`{"theme":"solarized-gruvbox"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Theme != "" {
		t.Errorf("Theme = %q, want empty", got.Theme)
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path := filepath.Join(xdg, "tmux-peeker", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("Load() error = %v, want error containing config path", err)
	}
}
