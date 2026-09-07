package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/lunemis/mux/tmux"
)

func TestLayoutDimensions(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "editor", WindowCount: 1, Created: time.Now().Add(-2 * time.Hour), Attached: true, Directory: "/Users/test/workspace/project1"},
		{Name: "dev-server", WindowCount: 2, Created: time.Now().Add(-24 * time.Hour), Attached: false, Directory: "/Users/test/workspace/project2"},
		{Name: "deploy", WindowCount: 1, Created: time.Now().Add(-48 * time.Hour), Attached: false, Directory: "/Users/test/workspace/project3"},
	}

	widths := []int{10, 40, 80, 120, 160, 200}
	heights := []int{10, 12, 20, 30, 40, 50}

	for _, w := range widths {
		for _, h := range heights {
			t.Run("", func(t *testing.T) {
				m := NewModel()
				m.width = w
				m.height = h
				m.sessions = sessions
				m.filtered = sessions
				m.rebuildItems()

				output := m.viewMain()
				lines := strings.Split(output, "\n")
				if len(lines) > h {
					t.Errorf("w=%d h=%d: output has %d lines, exceeds terminal height %d", w, h, len(lines), h)
				}
			})
		}
	}
}

func TestSessionRowDisplaysGenericCommand(t *testing.T) {
	row := ansi.Strip(formatSessionRow(tmux.Session{
		Name:          "work",
		Created:       time.Now(),
		ActiveCommand: "nvim",
	}, false, false, 60))

	if !strings.Contains(row, "nvim") {
		t.Fatalf("session row = %q, want generic pane command", row)
	}
}

func TestOverlayCenteredPreservesFullscreenCanvas(t *testing.T) {
	const width, height = 20, 7
	backgroundLine := strings.Repeat("b", width)
	background := strings.Repeat(backgroundLine+"\n", height-1) + backgroundLine
	foreground := "╭────╮\n│pick│\n╰────╯"

	output := ansi.Strip(overlayCentered(background, foreground, width, height))
	lines := strings.Split(output, "\n")
	if len(lines) != height {
		t.Fatalf("overlay lines = %d, want %d", len(lines), height)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != width {
			t.Errorf("line %d width = %d, want %d", i, got, width)
		}
	}
	if !strings.Contains(output, "│pick│") {
		t.Error("centered foreground is missing")
	}
	if lines[0] != backgroundLine || lines[height-1] != backgroundLine {
		t.Error("background outside the centered overlay was not preserved")
	}
}

func TestOverlayCenteredHandlesANSIAndWideCharacters(t *testing.T) {
	const width, height = 18, 5
	background := "界界界界界界界界界\n" + strings.Repeat("x\n", height-2) + strings.Repeat("y", width)
	foreground := titleStyle.Render("╭─ ◆ ─╮\n│ mux │\n╰─────╯")

	output := overlayCentered(background, foreground, width, height)
	for i, line := range strings.Split(output, "\n") {
		if got := ansi.StringWidth(line); got != width {
			t.Errorf("line %d width = %d, want %d", i, got, width)
		}
	}
	if !strings.Contains(ansi.Strip(output), "mux") {
		t.Error("styled overlay content is missing")
	}
}

func TestViewMainCompositesSelectorOverFullscreenPreview(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 20
	m.sessions = []tmux.Session{{
		Name: "selection-marker", Created: time.Now(), Directory: "/tmp/project",
	}}
	m.applyFilter()
	m.previewKey = previewKeyForItem(*m.currentItem())
	m.previewContent = "preview-marker"

	output := ansi.Strip(m.viewMain())
	if !strings.Contains(output, "preview-marker") || !strings.Contains(output, "tmux sessions") {
		t.Fatal("fullscreen view should contain both preview and selector")
	}
	if strings.Contains(output, "navigate") {
		t.Error("help should be hidden until requested")
	}
	lines := strings.Split(output, "\n")
	if len(lines) != m.height {
		t.Errorf("output lines = %d, want terminal height %d", len(lines), m.height)
	}
	if strings.TrimSpace(lines[0]) != "tmux session picker" {
		t.Errorf("top row = %q, want centered contextual title", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "preview-marker") {
		t.Errorf("bottom row lost bottom-left preview content: %q", lines[len(lines)-1])
	}
}

func TestHelpCardReplacesSelectorButPreservesPreview(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 20
	m.sessions = []tmux.Session{{Name: "work", Directory: "/tmp/project"}}
	m.applyFilter()
	m.previewKey = previewKeyForItem(*m.currentItem())
	m.previewContent = "preview-marker"
	m.helpVisible = true

	output := ansi.Strip(m.viewMain())
	if !strings.Contains(output, "preview-marker") || !strings.Contains(output, "navigate") {
		t.Fatal("help view should preserve the preview and show contextual commands")
	}
	if strings.Contains(output, "tmux sessions (1)") {
		t.Error("help card should replace rather than stack on the selector")
	}
}

func TestModalCompositesOverFullscreenPreview(t *testing.T) {
	m := NewModel()
	m.width = 80
	m.height = minimumSwitcherHeight
	m.mode = modeFilter
	m.filterMod = newFilterModel("needle")

	output := ansi.Strip(m.View())
	if !strings.Contains(output, "needle") {
		t.Error("filter modal should be visible over the preview")
	}
	if lines := strings.Count(output, "\n") + 1; lines != m.height {
		t.Errorf("output lines = %d, want terminal height %d", lines, m.height)
	}
}

func TestSessionListScrolling(t *testing.T) {
	sessions := make([]tmux.Session, 20)
	for i := range sessions {
		sessions[i] = tmux.Session{
			Name:        fmt.Sprintf("session-%02d", i),
			WindowCount: 1,
			Created:     time.Now(),
		}
	}

	width := 60
	height := 10

	out := renderSessionList(sessions, 0, "", width, height)
	if !strings.Contains(out, "session-00") {
		t.Error("cursor=0: expected session-00 to be visible")
	}

	out = renderSessionList(sessions, 15, "", width, height)
	if !strings.Contains(out, "session-15") {
		t.Error("cursor=15: expected session-15 to be visible")
	}
	if strings.Contains(out, "session-00") {
		t.Error("cursor=15: expected session-00 to be scrolled out")
	}

	out = renderSessionList(sessions, 19, "", width, height)
	if !strings.Contains(out, "session-19") {
		t.Error("cursor=19: expected session-19 to be visible")
	}
}
