package ui

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/aemonge/tmux-peeker/theme"
	"github.com/aemonge/tmux-peeker/tmux"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	foreground := titleStyle.Render("╭─ ◆ ─╮\n│ peek │\n╰─────╯")

	output := overlayCentered(background, foreground, width, height)
	for i, line := range strings.Split(output, "\n") {
		if got := ansi.StringWidth(line); got != width {
			t.Errorf("line %d width = %d, want %d", i, got, width)
		}
	}
	if !strings.Contains(ansi.Strip(output), "peek") {
		t.Error("styled overlay content is missing")
	}
}

func TestSelectorAndHelpCardsApplyExpectedBackgroundToEveryCell(t *testing.T) {
	useSolarizedTrueColor(t)

	surface := rgb{r: 251, g: 241, b: 199}
	selected := rgb{r: 213, g: 196, b: 161}
	empty := NewModel()
	empty.width = 100
	empty.height = 20

	populated := NewModel()
	populated.width = 100
	populated.height = 20
	populated.sessions = []tmux.Session{
		{Name: "work", Created: time.Now()},
		{Name: "notes", Created: time.Now()},
	}
	populated.applyFilter()

	window := tmux.Window{Index: 1, Name: "editor"}
	pane := tmux.Pane{Index: 2, Command: "nvim"}
	tests := []struct {
		name       string
		rendered   string
		background []rgb
		width      int
		height     int
	}{
		{
			name: "empty selector", rendered: renderSwitcherSelector(&empty),
			background: []rgb{surface}, width: switcherWidth(empty.width),
			height: switcherRows(0, empty.height) + 2,
		},
		{
			name: "help", rendered: renderSwitcherHelp(populated.keyMap, populated.width, populated.height),
			background: []rgb{surface}, width: switcherWidth(populated.width), height: 11,
		},
		{
			name: "narrow help", rendered: renderSwitcherHelp(populated.keyMap, 76, populated.height),
			background: []rgb{surface}, width: switcherWidth(76), height: 11,
		},
		{name: "session row", rendered: formatSessionRow(populated.sessions[1], false, false, 48), background: []rgb{surface}, width: 48, height: 1},
		{name: "window row", rendered: formatWindowRow(&window, false, false, 48), background: []rgb{surface}, width: 48, height: 1},
		{name: "pane row", rendered: formatPaneRow(&pane, false, 48), background: []rgb{surface}, width: 48, height: 1},
		{name: "selected session", rendered: formatSessionRow(populated.sessions[0], false, true, 48), background: []rgb{selected}, width: 48, height: 1},
		{name: "selected window", rendered: formatWindowRow(&window, false, true, 48), background: []rgb{selected}, width: 48, height: 1},
		{name: "selected pane", rendered: formatPaneRow(&pane, true, 48), background: []rgb{selected}, width: 48, height: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertRenderedDimensions(t, test.rendered, test.width, test.height)
			assertEveryVisibleCellUsesBackground(t, test.rendered, test.background...)
		})
	}

	t.Run("composed selector roles", func(t *testing.T) {
		rendered := renderSwitcherSelector(&populated)
		width := switcherWidth(populated.width)
		height := switcherRows(len(populated.sessions), populated.height) + 2
		assertRenderedDimensions(t, rendered, width, height)
		lines := strings.Split(rendered, "\n")
		assertEveryVisibleCellUsesBackground(t, lines[0], surface)
		assertEveryVisibleCellUsesBackground(t, ansi.Cut(lines[1], 0, 1), surface)
		assertEveryVisibleCellUsesBackground(t, ansi.Cut(lines[1], 1, width-1), selected)
		assertEveryVisibleCellUsesBackground(t, ansi.Cut(lines[1], width-1, width), surface)
		for _, line := range lines[2:] {
			assertEveryVisibleCellUsesBackground(t, line, surface)
		}
	})
}

func TestTitledBorderUsesSeparatorColorOnEveryEdge(t *testing.T) {
	useSolarizedTrueColor(t)

	const width = 10
	rendered := drawTitledBorder("title", surfaceSpaces(width-2), width, 1)
	assertRenderedDimensions(t, rendered, width, 3)
	lines := strings.Split(rendered, "\n")
	separator := rgb{r: 7, g: 102, b: 120}
	assertEveryVisibleCellUsesForeground(t, lines[0], separator)
	assertEveryVisibleCellUsesForeground(t, ansi.Cut(lines[1], 0, 1), separator)
	assertEveryVisibleCellUsesForeground(t, ansi.Cut(lines[1], width-1, width), separator)
	assertEveryVisibleCellUsesForeground(t, lines[2], separator)
}

func TestSurfaceDoesNotApplyToPreviewCanvasOrModal(t *testing.T) {
	useSolarizedTrueColor(t)

	m := NewModel()
	m.width = 80
	m.height = 20
	m.previewContent = "preview"
	surface := rgb{r: 251, g: 241, b: 199}
	for name, rendered := range map[string]string{
		"preview canvas": m.previewBackground(),
		"modal":          renderModal("content"),
	} {
		t.Run(name, func(t *testing.T) {
			assertNoVisibleCellUsesBackground(t, rendered, surface)
		})
	}
}

func useSolarizedTrueColor(t *testing.T) {
	t.Helper()
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(0) // termenv.TrueColor
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previousProfile)
		UseTheme(theme.Default)
	})

	selectedTheme, err := theme.Get("solarized-gruvbox")
	if err != nil {
		t.Fatal(err)
	}
	UseTheme(selectedTheme)
}

type rgb struct {
	r int
	g int
	b int
}

func assertRenderedDimensions(t *testing.T, rendered string, width, height int) {
	t.Helper()
	lines := strings.Split(rendered, "\n")
	if len(lines) != height {
		t.Fatalf("rendered height = %d, want %d", len(lines), height)
	}
	for index, line := range lines {
		if got := ansi.StringWidth(line); got != width {
			t.Fatalf("line %d width = %d, want %d", index, got, width)
		}
	}
}

func assertEveryVisibleCellUsesBackground(t *testing.T, rendered string, expected ...rgb) {
	t.Helper()
	visibleCells := 0
	visitVisibleBackgrounds(rendered, func(offset int, r rune, background *rgb) {
		visibleCells++
		for _, color := range expected {
			if background != nil && approximatelyEqualRGB(*background, color) {
				return
			}
		}
		t.Fatalf("cell %q at byte %d has background %v, want one of %v; rendered:\n%s", r, offset, background, expected, ansi.Strip(rendered))
	})
	if visibleCells == 0 {
		t.Fatal("rendered output has no visible cells")
	}
}

func assertEveryVisibleCellUsesForeground(t *testing.T, rendered string, expected rgb) {
	t.Helper()
	visibleCells := 0
	var foreground *rgb
	for offset := 0; offset < len(rendered); {
		if rendered[offset] == '\x1b' && offset+1 < len(rendered) && rendered[offset+1] == '[' {
			end := strings.IndexByte(rendered[offset+2:], 'm')
			if end >= 0 {
				applyForegroundSGR(rendered[offset+2:offset+2+end], &foreground)
				offset += end + 3
				continue
			}
		}

		r, size := utf8.DecodeRuneInString(rendered[offset:])
		if r != '\n' {
			visibleCells++
			if foreground == nil || !approximatelyEqualRGB(*foreground, expected) {
				t.Fatalf("cell %q at byte %d has foreground %v, want %v; rendered:\n%s", r, offset, foreground, expected, ansi.Strip(rendered))
			}
		}
		offset += size
	}
	if visibleCells == 0 {
		t.Fatal("rendered output has no visible cells")
	}
}

func assertNoVisibleCellUsesBackground(t *testing.T, rendered string, forbidden rgb) {
	t.Helper()
	visibleCells := 0
	visitVisibleBackgrounds(rendered, func(offset int, r rune, background *rgb) {
		visibleCells++
		if background != nil && approximatelyEqualRGB(*background, forbidden) {
			t.Fatalf("cell %q at byte %d unexpectedly uses surface background; rendered:\n%s", r, offset, ansi.Strip(rendered))
		}
	})
	if visibleCells == 0 {
		t.Fatal("rendered output has no visible cells")
	}
}

func visitVisibleBackgrounds(rendered string, visit func(offset int, r rune, background *rgb)) {
	var background *rgb
	for offset := 0; offset < len(rendered); {
		if rendered[offset] == '\x1b' && offset+1 < len(rendered) && rendered[offset+1] == '[' {
			end := strings.IndexByte(rendered[offset+2:], 'm')
			if end >= 0 {
				applyBackgroundSGR(rendered[offset+2:offset+2+end], &background)
				offset += end + 3
				continue
			}
		}

		r, size := utf8.DecodeRuneInString(rendered[offset:])
		if r != '\n' {
			visit(offset, r, background)
		}
		offset += size
	}
}

func approximatelyEqualRGB(left, right rgb) bool {
	differences := 0
	for _, delta := range []int{left.r - right.r, left.g - right.g, left.b - right.b} {
		if absInt(delta) > 1 {
			return false
		}
		if delta != 0 {
			differences++
		}
	}
	return differences <= 1
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func applyForegroundSGR(sequence string, foreground **rgb) {
	parameters := strings.Split(sequence, ";")
	for i := 0; i < len(parameters); i++ {
		switch parameters[i] {
		case "", "0", "39":
			*foreground = nil
		case "38":
			if i+4 >= len(parameters) || parameters[i+1] != "2" {
				continue
			}
			r, errR := strconv.Atoi(parameters[i+2])
			g, errG := strconv.Atoi(parameters[i+3])
			b, errB := strconv.Atoi(parameters[i+4])
			if errR == nil && errG == nil && errB == nil {
				color := rgb{r: r, g: g, b: b}
				*foreground = &color
			}
			i += 4
		}
	}
}

func applyBackgroundSGR(sequence string, background **rgb) {
	parameters := strings.Split(sequence, ";")
	for i := 0; i < len(parameters); i++ {
		switch parameters[i] {
		case "", "0", "49":
			*background = nil
		case "48":
			if i+4 >= len(parameters) || parameters[i+1] != "2" {
				continue
			}
			r, errR := strconv.Atoi(parameters[i+2])
			g, errG := strconv.Atoi(parameters[i+3])
			b, errB := strconv.Atoi(parameters[i+4])
			if errR == nil && errG == nil && errB == nil {
				color := rgb{r: r, g: g, b: b}
				*background = &color
			}
			i += 4
		}
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
