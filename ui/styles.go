package ui

import (
	"strings"

	"github.com/aemonge/tmux-peeker/theme"
	"github.com/charmbracelet/lipgloss"
)

var (
	applyBackground bool

	// Colors
	colorBackground lipgloss.Color
	colorPrimary    lipgloss.Color
	colorAccent     lipgloss.Color
	colorDanger     lipgloss.Color
	colorMuted      lipgloss.Color
	colorBorder     lipgloss.Color
	colorSeparator  lipgloss.Color
	colorSelected   lipgloss.Color
	colorCursor     lipgloss.Color
	colorText       lipgloss.Color

	// Styles
	titleStyle      lipgloss.Style
	helpStyle       lipgloss.Style
	helpKeyStyle    lipgloss.Style
	errorStyle      lipgloss.Style
	inputLabelStyle lipgloss.Style
)

func init() {
	UseTheme(theme.Default)
}

// UseTheme applies a loaded theme to subsequent UI rendering.
func UseTheme(value theme.Theme) {
	background := strings.TrimSpace(value.Colors.Background)
	applyBackground = background != "" && !strings.EqualFold(background, "NONE")
	colorBackground = lipgloss.Color("")
	if applyBackground {
		colorBackground = lipgloss.Color(background)
	}
	colorPrimary = lipgloss.Color(value.Colors.Primary)
	colorAccent = lipgloss.Color(value.Colors.Accent)
	colorDanger = lipgloss.Color(value.Colors.Danger)
	colorMuted = lipgloss.Color(value.Colors.Muted)
	colorBorder = lipgloss.Color(value.Colors.Border)
	colorSeparator = lipgloss.Color(value.Colors.Separator)
	colorSelected = lipgloss.Color(value.Colors.Selected)
	colorCursor = lipgloss.Color(value.Colors.Cursor)
	colorText = lipgloss.Color(value.Colors.Text)

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	helpStyle = lipgloss.NewStyle().Foreground(colorMuted)
	helpKeyStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	inputLabelStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
}
