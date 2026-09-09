package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfiguredThemeNameReadsXDGConfig(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv(themeEnv, "")
	path := filepath.Join(xdg, "tmux-peeker", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"solarized-gruvbox"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := configuredThemeName()
	if err != nil {
		t.Fatalf("configuredThemeName() error = %v", err)
	}
	if got != "solarized-gruvbox" {
		t.Errorf("configuredThemeName() = %q, want solarized-gruvbox", got)
	}
}

func TestConfiguredKeyMapReadsXDGConfig(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path := filepath.Join(xdg, "tmux-peeker", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"keybindings":{"list":{"up":["w"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := configuredKeyMap()
	if err != nil {
		t.Fatalf("configuredKeyMap() error = %v", err)
	}
	if !got.Matches("list", "up", "w") {
		t.Error("configuredKeyMap() does not use configured list.up binding")
	}
	if got.Matches("list", "up", "k") {
		t.Error("configuredKeyMap() still uses replaced default list.up binding")
	}
}

func TestConfiguredThemeNameEnvironmentOverridesConfig(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv(themeEnv, "default")
	path := filepath.Join(xdg, "tmux-peeker", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"solarized-gruvbox"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := configuredThemeName()
	if err != nil {
		t.Fatalf("configuredThemeName() error = %v", err)
	}
	if got != "default" {
		t.Errorf("configuredThemeName() = %q, want default", got)
	}
}

func TestConfiguredThemeNameIgnoresUpstreamEnvironment(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(themeEnv, "")
	t.Setenv("MUX_THEME", "solarized-gruvbox")

	got, err := configuredThemeName()
	if err != nil {
		t.Fatalf("configuredThemeName() error = %v", err)
	}
	if got != "default" {
		t.Errorf("configuredThemeName() = %q, want default", got)
	}
}

func TestConfigureTheme(t *testing.T) {
	if err := configureTheme("solarized-gruvbox"); err != nil {
		t.Fatalf("configureTheme() error = %v", err)
	}
	t.Cleanup(func() {
		if err := configureTheme("default"); err != nil {
			t.Errorf("restore default theme: %v", err)
		}
	})

	err := configureTheme("unknown")
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("configureTheme(unknown) error = %v, want unknown theme error", err)
	}
}
