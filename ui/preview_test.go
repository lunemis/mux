package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderPreviewFillsCanvasWithoutChrome(t *testing.T) {
	const width, height = 24, 6
	output := ansi.Strip(renderPreview("preview output", width, height))
	lines := strings.Split(output, "\n")

	if len(lines) != height {
		t.Fatalf("preview lines = %d, want %d", len(lines), height)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != width {
			t.Errorf("line %d width = %d, want %d", i, got, width)
		}
	}
	if strings.ContainsAny(output, "╭╮╰╯│") {
		t.Errorf("edge-to-edge preview contains border chrome: %q", output)
	}
	if strings.Contains(output, "[ pi ]") || strings.Contains(output, "──") {
		t.Errorf("edge-to-edge preview contains legacy title chrome: %q", output)
	}
}

func TestRenderPreviewAnchorsCapturedOutputBottomLeft(t *testing.T) {
	output := ansi.Strip(renderPreview("prompt", 24, 6))
	lines := strings.Split(output, "\n")
	if !strings.HasPrefix(lines[len(lines)-1], "prompt") {
		t.Errorf("bottom row = %q, want prompt anchored at left", lines[len(lines)-1])
	}
	if strings.Contains(strings.Join(lines[:len(lines)-1], "\n"), "prompt") {
		t.Error("prompt should render only on the bottom row")
	}
}

func TestRenderPreviewCropsTopAndRightWithoutWrapping(t *testing.T) {
	output := ansi.Strip(renderPreview("old\nleft-edge-right\nlatest", 10, 2))
	if strings.Contains(output, "old") {
		t.Error("preview should crop the oldest row from the top")
	}
	if strings.Contains(output, "right") {
		t.Error("preview should crop overflowing text from the right")
	}
	if !strings.Contains(output, "left-edge-") || !strings.Contains(output, "latest") {
		t.Error("preview should preserve bottom rows and their left edge")
	}
	if lines := strings.Count(output, "\n") + 1; lines != 2 {
		t.Errorf("preview lines = %d, want 2 without wrapping", lines)
	}
}

func TestRenderPreviewEmptyCanvas(t *testing.T) {
	const width, height = 12, 4
	output := renderPreview("", width, height)
	for i, line := range strings.Split(output, "\n") {
		if got := ansi.StringWidth(line); got != width {
			t.Errorf("empty line %d width = %d, want %d", i, got, width)
		}
	}
	if lines := strings.Count(output, "\n") + 1; lines != height {
		t.Errorf("empty preview lines = %d, want %d", lines, height)
	}
}
