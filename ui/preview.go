package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lunemis/mux/tmux"
)

func renderPreview(item *listItem, captured string, width, height int, tokenUsage *tmux.TokenUsage) string {
	innerWidth := width - 2
	innerHeight := height - 2

	if item == nil {
		lines := make([]string, innerHeight)
		mid := innerHeight / 2
		msg := "No session selected"
		for i := range lines {
			if i == mid {
				lines[i] = padOrTruncate(centerText(msg, innerWidth), innerWidth)
			} else {
				lines[i] = strings.Repeat(" ", innerWidth)
			}
		}
		content := strings.Join(lines, "\n")
		return drawBorder(content, width, innerHeight)
	}

	session := item.session
	// Header: build text first, append styled suffixes after padding
	// to prevent ANSI codes and ambiguous-width icons from clipping.
	badge := aiLabelPlain(session.ActiveCommand)

	branchInfo := ""
	branchStyled := ""
	if session.GitBranch != "" {
		prefix := "⌥"
		if session.IsWorktree {
			prefix = "⌥⌥"
		}
		branchText := prefix + " " + session.GitBranch
		branchInfo = "  " + branchText
		branchStyled = "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(branchText)
	}

	label := previewLabel(item)
	headerText := fmt.Sprintf("[ %s ]  %s", label, shortenPath(session.Directory))
	headerWidth := innerWidth - len(badge.text) - badge.extraWidth - len(branchInfo)
	if headerWidth < 10 {
		headerWidth = 10
	}
	headerPadded := padOrTruncate(headerText, headerWidth)

	headerStyled := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(headerPadded) + badge.styled + branchStyled
	separator := lipgloss.NewStyle().Foreground(colorBorder).Render(
		strings.Repeat("─", innerWidth))

	// Token usage line (optional)
	var tokenLine string
	if tokenUsage != nil {
		tokenLine = formatTokenLine(tokenUsage, innerWidth)
	}

	// Header block: header, [token], separator — truncated when the panel is
	// shorter than the header itself.
	header := []string{headerStyled}
	if tokenLine != "" {
		header = append(header, tokenLine)
	}
	header = append(header, separator)
	if len(header) > innerHeight {
		header = header[:innerHeight]
	}
	contentLines := innerHeight - len(header)

	capLines := strings.Split(captured, "\n")
	// Keep last N lines (most recent output)
	if len(capLines) > contentLines {
		capLines = capLines[len(capLines)-contentLines:]
	}

	allLines := make([]string, innerHeight)
	copy(allLines, header)
	for i := 0; i < contentLines; i++ {
		if i < len(capLines) {
			allLines[len(header)+i] = padOrTruncate(capLines[i], innerWidth)
		} else {
			allLines[len(header)+i] = strings.Repeat(" ", innerWidth)
		}
	}

	content := strings.Join(allLines, "\n")
	return drawBorder(content, width, innerHeight)
}

// previewLabel returns the header label for the previewed target. For session
// rows it's just the session name; for windows/panes it appends the
// hierarchy and command/window name.
func previewLabel(it *listItem) string {
	switch it.kind {
	case itemWindow:
		return fmt.Sprintf("%s · %d:%s", it.session.Name, it.window.Index, it.window.Name)
	case itemPane:
		return fmt.Sprintf("%s · %d.%d %s", it.session.Name, it.window.Index, it.pane.Index, it.pane.Command)
	default:
		return it.session.Name
	}
}

// labelInfo holds both the styled and plain-text versions of a badge,
// plus extra width to compensate for ambiguous-width Unicode characters
// that terminals render as 2 cells but ansi.StringWidth measures as 1.
type labelInfo struct {
	text       string // plain text for width calculation (e.g. "  ✦ claude")
	styled     string // ANSI-styled version for display
	extraWidth int    // extra cells for ambiguous-width chars (1 per icon)
}

func aiLabelPlain(cmd string) labelInfo {
	tool, ok := tmux.LookupAITool(cmd)
	if !ok {
		return labelInfo{}
	}
	text := "  " + tool.Icon + " " + tool.Name
	styled := "  " + lipgloss.NewStyle().Foreground(lipgloss.Color(tool.Color)).Bold(true).Render(tool.Icon+" "+tool.Name)
	return labelInfo{text: text, styled: styled, extraWidth: 1}
}

func formatTokenLine(u *tmux.TokenUsage, width int) string {
	text := fmt.Sprintf("  %s in / %s out  ~$%.2f",
		tmux.FormatTokens(u.InputTokens),
		tmux.FormatTokens(u.OutputTokens),
		u.TotalCost)
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(
		padOrTruncate(text, width))
	return styled
}

func shortenPath(path string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if strings.HasPrefix(path, home) {
			path = "~" + path[len(home):]
		}
	}
	if len(path) > maxPathDisplay {
		path = "..." + path[len(path)-maxPathDisplay+3:]
	}
	return path
}
