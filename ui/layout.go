package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const minimumSwitcherHeight = 12

// padOrTruncate ensures a string is exactly `width` visible characters
func padOrTruncate(s string, width int) string {
	w := ansi.StringWidth(s)
	if w > width {
		return ansi.Truncate(s, width, "")
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

func truncateAndCenter(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(s) > width {
		s = ansi.Truncate(s, width, "")
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, s)
}

// fixedBox takes rendered content and forces it to exactly width x height visible area.
// It splits by newlines, truncates/pads each line to width, and truncates/pads to height lines.
func fixedBox(content string, width, height int) string {
	lines := strings.Split(content, "\n")

	result := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			result[i] = padOrTruncate(lines[i], width)
		} else {
			result[i] = strings.Repeat(" ", width)
		}
	}
	return strings.Join(result, "\n")
}

// overlayCentered composites foreground over background at terminal-cell
// boundaries. ANSI-aware cuts preserve styling and wide-character alignment.
func overlayCentered(background, foreground string, width, height int) string {
	backgroundLines := strings.Split(fixedBox(background, width, height), "\n")
	foregroundLines := strings.Split(foreground, "\n")
	foregroundWidth := 0
	for _, line := range foregroundLines {
		foregroundWidth = max(foregroundWidth, ansi.StringWidth(line))
	}
	foregroundWidth = min(foregroundWidth, width)
	foregroundHeight := min(len(foregroundLines), height)
	x := max(0, (width-foregroundWidth)/2)
	y := max(0, (height-foregroundHeight)/2)

	for i := 0; i < foregroundHeight; i++ {
		base := backgroundLines[y+i]
		overlay := padOrTruncate(foregroundLines[i], foregroundWidth)
		left := ansi.Cut(base, 0, x)
		right := ansi.Cut(base, x+foregroundWidth, width)
		backgroundLines[y+i] = padOrTruncate(left+overlay+right, width)
	}
	return strings.Join(backgroundLines, "\n")
}

// drawTitledBorder wraps fixed-height content in a rounded border whose top
// edge carries a compact context title.
func drawTitledBorder(title, content string, width, height int) string {
	innerWidth := max(0, width-2)
	label := " " + title + " "
	label = ansi.Truncate(label, innerWidth, "")
	top := "╭" + label + strings.Repeat("─", max(0, innerWidth-ansi.StringWidth(label))) + "╮"

	lines := strings.Split(content, "\n")
	borderStyle := lipgloss.NewStyle().Foreground(colorBorder).Background(colorSurface)
	titleBorderStyle := lipgloss.NewStyle().Foreground(colorSeparator).Background(colorSurface)
	result := make([]string, 0, height+2)
	result = append(result, titleBorderStyle.Render(top))
	for i := 0; i < height; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		result = append(result, borderStyle.Render("│")+padOrTruncate(line, innerWidth)+borderStyle.Render("│"))
	}
	result = append(result, borderStyle.Render("╰"+strings.Repeat("─", innerWidth)+"╯"))
	return strings.Join(result, "\n")
}

// drawBorder wraps content lines with a rounded border
func drawBorder(content string, width, height int) string {
	innerWidth := width - 2
	lines := strings.Split(content, "\n")

	// Build bordered output
	result := make([]string, 0, height+2)

	// Top border
	result = append(result, "╭"+strings.Repeat("─", innerWidth)+"╮")

	// Content lines (pad/truncate to exactly height)
	for i := 0; i < height; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		line = padOrTruncate(line, innerWidth)
		result = append(result, "│"+line+"│")
	}

	// Bottom border
	result = append(result, "╰"+strings.Repeat("─", innerWidth)+"╯")

	borderStyle := lipgloss.NewStyle().Foreground(colorBorder)
	result[0] = borderStyle.Render(result[0])
	for i := 1; i < len(result)-1; i++ {
		line := strings.TrimSuffix(strings.TrimPrefix(result[i], "│"), "│")
		result[i] = borderStyle.Render("│") + line + borderStyle.Render("│")
	}
	result[len(result)-1] = borderStyle.Render(result[len(result)-1])
	return strings.Join(result, "\n")
}
