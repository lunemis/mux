package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/lunemis/mux/tmux"
)

func TestShortenPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{home + "/projects/foo", "~/projects/foo"},
		{home, "~"},
		{"/tmp/other", "/tmp/other"},
		{"", ""},
	}

	for _, tt := range tests {
		got := shortenPath(tt.input)
		if got != tt.want {
			t.Errorf("shortenPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShortenPathTruncatesLong(t *testing.T) {
	long := "/very/long/path/that/exceeds/thirty/five/characters/definitely"
	got := shortenPath(long)
	if len(got) > 35 {
		t.Errorf("shortenPath should truncate to 35 chars, got len=%d: %q", len(got), got)
	}
	if !strings.HasPrefix(got, "...") {
		t.Errorf("truncated path should start with '...', got %q", got)
	}
}

func TestAiLabelPlain(t *testing.T) {
	// Known commands should return non-empty
	for _, cmd := range []string{"claude", "codex", "aider", "gemini"} {
		info := aiLabelPlain(cmd)
		if info.styled == "" {
			t.Errorf("aiLabelPlain(%q) returned empty styled", cmd)
		}
		if info.text == "" {
			t.Errorf("aiLabelPlain(%q) returned empty text", cmd)
		}
		if info.extraWidth != 1 {
			t.Errorf("aiLabelPlain(%q) extraWidth = %d, want 1", cmd, info.extraWidth)
		}
	}
	// Unknown commands should return empty
	info := aiLabelPlain("bash")
	if info.styled != "" {
		t.Errorf("aiLabelPlain(%q) styled = %q, want empty", "bash", info.styled)
	}
}

func TestRenderPreviewNilSession(t *testing.T) {
	output := renderPreview(nil, "", 40, 10, nil)
	if !strings.Contains(output, "No session selected") {
		t.Error("nil session should show 'No session selected'")
	}
}

func TestRenderPreviewSmallHeights(t *testing.T) {
	item := &listItem{
		kind: itemSession,
		session: &tmux.Session{
			Name:          "work",
			Directory:     "/tmp/work",
			ActiveCommand: "claude",
		},
	}
	usages := map[string]*tmux.TokenUsage{
		"nil-usage": nil,
		"with-usage": {
			InputTokens:  100,
			OutputTokens: 200,
			TotalCost:    1.23,
		},
	}
	captured := "line1\nline2\nline3\nline4\nline5"

	for name, usage := range usages {
		for height := 3; height <= 12; height++ {
			gotLines := func() int {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("%s height=%d: renderPreview panicked: %v", name, height, r)
					}
				}()
				out := renderPreview(item, captured, 40, height, usage)
				return len(strings.Split(out, "\n"))
			}()
			if gotLines != height {
				t.Errorf("%s height=%d: got %d lines, want %d", name, height, gotLines, height)
			}
		}
	}
}
