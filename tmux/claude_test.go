package tmux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncodePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Users/foo/bar", "-Users-foo-bar"},
		{"/home/user/project", "-home-user-project"},
		// Claude Code encodes "." as "-" too (e.g. .worktrees, .config).
		{"/Users/koma/proj/.worktrees/x", "-Users-koma-proj--worktrees-x"},
		{"/Users/foo/my.app", "-Users-foo-my-app"},
		{"", ""},
	}
	for _, tt := range tests {
		got := encodePath(tt.input)
		if got != tt.want {
			t.Errorf("encodePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{500, "500"},
		{1500, "1.5k"},
		{45200, "45.2k"},
		{1500000, "1.5M"},
		{0, "0"},
	}
	for _, tt := range tests {
		got := FormatTokens(tt.input)
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEstimateCost(t *testing.T) {
	u := &TokenUsage{
		InputTokens:  1_000_000,
		OutputTokens: 100_000,
		CacheRead:    500_000,
		CacheWrite:   200_000,
	}
	cost := estimateCost(u)
	// Input: 1M * $15/1M = $15
	// Output: 0.1M * $75/1M = $7.5
	// CacheRead: 0.5M * $1.5/1M = $0.75
	// CacheWrite: 0.2M * $18.75/1M = $3.75
	// Total = $27.0
	expected := 27.0
	if cost < expected-0.01 || cost > expected+0.01 {
		t.Errorf("estimateCost() = %f, want ~%f", cost, expected)
	}
}

func TestParseTokenUsage(t *testing.T) {
	// Create a temp JSONL file with sample data
	dir := t.TempDir()
	path := filepath.Join(dir, "test-session.jsonl")

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"hello"}}`,
		`{"type":"assistant","message":{"model":"claude-opus-4-6","role":"assistant","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":200,"cache_creation_input_tokens":300}}}`,
		`{"type":"file-history-snapshot","snapshot":{}}`,
		`{"type":"assistant","message":{"model":"claude-opus-4-6","role":"assistant","usage":{"input_tokens":150,"output_tokens":75,"cache_read_input_tokens":100,"cache_creation_input_tokens":0}}}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	usage, err := parseTokenUsage(path)
	if err != nil {
		t.Fatal(err)
	}

	if usage.InputTokens != 250 {
		t.Errorf("InputTokens = %d, want 250", usage.InputTokens)
	}
	if usage.OutputTokens != 125 {
		t.Errorf("OutputTokens = %d, want 125", usage.OutputTokens)
	}
	if usage.CacheRead != 300 {
		t.Errorf("CacheRead = %d, want 300", usage.CacheRead)
	}
	if usage.CacheWrite != 300 {
		t.Errorf("CacheWrite = %d, want 300", usage.CacheWrite)
	}
	if usage.TotalCost <= 0 {
		t.Error("TotalCost should be positive")
	}
}

func TestParseTokenUsageMissingFile(t *testing.T) {
	_, err := parseTokenUsage("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// Real-world JSONL lines can exceed 1MB (large tool results); they must still
// be parsed rather than silently truncating the aggregate.
func TestParseTokenUsageLargeLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.jsonl")

	pad := strings.Repeat("A", 2*1024*1024)
	lines := []string{
		`{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}},"pad":"` + pad + `"}`,
		`{"type":"assistant","message":{"usage":{"input_tokens":150,"output_tokens":75,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	usage, err := parseTokenUsage(path)
	if err != nil {
		t.Fatal(err)
	}
	if usage.InputTokens != 250 {
		t.Errorf("InputTokens = %d, want 250", usage.InputTokens)
	}
	if usage.OutputTokens != 125 {
		t.Errorf("OutputTokens = %d, want 125", usage.OutputTokens)
	}
}

// A line beyond the scanner cap must surface an error instead of returning a
// silently partial (and then cached) aggregate.
func TestParseTokenUsageOversizedLineReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oversized.jsonl")

	pad := strings.Repeat("A", maxScanTokenSize+1024)
	line := `{"type":"assistant","message":{"usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}},"pad":"` + pad + `"}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := parseTokenUsage(path); err == nil {
		t.Error("expected error for line exceeding scanner cap")
	}
}
