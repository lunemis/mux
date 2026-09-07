package ui

import (
	"testing"

	"github.com/aemonge/tmux-peeker/theme"
)

func TestUseThemeAppliesSemanticPalette(t *testing.T) {
	defer UseTheme(theme.Default)

	selected, err := theme.Get("solarized-gruvbox")
	if err != nil {
		t.Fatal(err)
	}
	UseTheme(selected)

	if applyBackground {
		t.Error("background NONE must not apply a terminal background")
	}
	if got := string(colorBackground); got != "" {
		t.Errorf("colorBackground = %q, want empty", got)
	}
	if got := string(colorText); got != "#3C3836" {
		t.Errorf("colorText = %q, want #3C3836", got)
	}
	if got := string(colorSeparator); got != "#076678" {
		t.Errorf("colorSeparator = %q, want #076678", got)
	}
	if got := string(colorSurface); got != "#FBF1C7" {
		t.Errorf("colorSurface = %q, want #FBF1C7", got)
	}
}
