package ui

import "strings"

// renderPreview renders captured terminal cells across the exact canvas. It
// keeps the bottom-left visible when the source is larger and pads above/right
// when it is smaller; preview content is never wrapped.
func renderPreview(captured string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	var capturedLines []string
	if captured != "" {
		capturedLines = strings.Split(captured, "\n")
	}
	if len(capturedLines) > height {
		capturedLines = capturedLines[len(capturedLines)-height:]
	}

	lines := make([]string, height)
	for i := range lines {
		lines[i] = strings.Repeat(" ", width)
	}
	start := height - len(capturedLines)
	for i, line := range capturedLines {
		lines[start+i] = padOrTruncate(line, width)
	}
	return strings.Join(lines, "\n")
}
