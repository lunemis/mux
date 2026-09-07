package tmux

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockRunner records calls and returns pre-configured responses.
type mockRunner struct {
	outputs     map[string]mockResult
	outputCalls []string
	runs        []string
}

type mockResult struct {
	out []byte
	err error
}

func newMockRunner() *mockRunner {
	return &mockRunner{outputs: make(map[string]mockResult)}
}

func (m *mockRunner) key(name string, args ...string) string {
	return name + " " + strings.Join(args, " ")
}

func (m *mockRunner) OnOutput(out []byte, err error, name string, args ...string) {
	m.outputs[m.key(name, args...)] = mockResult{out: out, err: err}
}

func (m *mockRunner) Output(name string, args ...string) ([]byte, error) {
	k := m.key(name, args...)
	m.outputCalls = append(m.outputCalls, k)
	if r, ok := m.outputs[k]; ok {
		return r.out, r.err
	}
	return nil, fmt.Errorf("mock: unexpected call: %s", k)
}

func (m *mockRunner) Run(name string, args ...string) error {
	m.runs = append(m.runs, m.key(name, args...))
	return nil
}

func withMock(t *testing.T, fn func(m *mockRunner)) {
	t.Helper()
	t.Setenv("TMUX_PANE", "")
	t.Setenv("MUX_ORIGIN_SESSION", "")
	m := newMockRunner()
	old := runner
	SetRunner(m)
	// Clear command cache to avoid cross-test interference
	cmdCacheMu.Lock()
	cmdCache = make(map[int]cachedCommand)
	cmdCacheMu.Unlock()
	defer func() { runner = old }()
	fn(m)
}

func TestListSessionsWithMock(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		t.Setenv("TMUX_PANE", "%9")
		now := time.Now().Unix()
		line1 := fmt.Sprintf("dev|2|%d|1|/home/user/dev|%d|bash|100", now-3600, now-3600)
		line2 := fmt.Sprintf("ai|1|%d|0|/home/user/ai|%d|claude|200", now-7200, now-120)
		out := line1 + "\n" + line2

		m.OnOutput([]byte(out), nil, "tmux", "list-sessions", "-F", listFormat)
		m.OnOutput([]byte("1 0\n"), nil,
			"ps", "-e", "-o", "pid=", "-o", "ppid=")
		m.OnOutput([]byte("ai\n"), nil,
			"tmux", "display-message", "-p", "-t", "%9", "#{session_name}")

		sessions, err := ListSessions()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(sessions))
		}
		// The current session is first and the UI initially highlights the previous session.
		if sessions[0].Name != "ai" || sessions[1].Name != "dev" {
			t.Errorf("session order = [%s %s], want [ai dev]", sessions[0].Name, sessions[1].Name)
		}
		if !sessions[0].Current || sessions[1].Current {
			t.Errorf("Current flags = [%t %t], want [true false]", sessions[0].Current, sessions[1].Current)
		}
		assertOutputCalls(t, m, []string{
			"tmux list-sessions -F " + listFormat,
			"ps -e -o pid= -o ppid=",
			"tmux display-message -p -t %9 #{session_name}",
		})
	})
}

func TestCurrentSessionNamePrefersPopupOrigin(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		t.Setenv("MUX_ORIGIN_SESSION", "origin")
		t.Setenv("TMUX_PANE", "%popup")

		got := currentSessionName()
		if got != "origin" {
			t.Errorf("currentSessionName() = %q, want origin", got)
		}
	})
}

func TestCurrentSessionNameUsesInvokingPane(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		t.Setenv("TMUX_PANE", "%42")
		m.OnOutput([]byte("work\n"), nil,
			"tmux", "display-message", "-p", "-t", "%42", "#{session_name}")

		got := currentSessionName()
		if got != "work" {
			t.Errorf("currentSessionName() = %q, want work", got)
		}
	})
}

func TestCurrentSessionNameOutsideTmuxIsEmpty(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		if got := currentSessionName(); got != "" {
			t.Errorf("currentSessionName() = %q, want empty", got)
		}
		if len(m.outputs) != 0 {
			t.Errorf("unexpected tmux output calls configured: %d", len(m.outputs))
		}
	})
}

func TestResolveCommandWithMock(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		// pgrep returns child PIDs
		m.OnOutput([]byte("42\n43\n"), nil, "pgrep", "-P", "100")
		// ps for PID 42 returns bash
		m.OnOutput([]byte("/bin/bash\n"), nil, "ps", "-o", "args=", "-p", "42")
		// ps for PID 43 returns claude
		m.OnOutput([]byte("/usr/local/bin/claude --help\n"), nil, "ps", "-o", "args=", "-p", "43")

		result := resolveCommand(100, "bash")
		if result != "claude" {
			t.Errorf("expected 'claude', got %q", result)
		}
	})
}

func TestCapturePaneWithMock(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		m.OnOutput([]byte("hello world\n\n"), nil, "tmux", "capture-pane", "-t", "test-session", "-p", "-e")

		content, err := CapturePane("test-session")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "hello world\n" {
			t.Errorf("expected one trailing blank row to be preserved, got %q", content)
		}
	})
}

func TestCreateSessionWithMock(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		if err := CreateSession("new-sess"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.runs) != 1 {
			t.Fatalf("expected 1 run call, got %d", len(m.runs))
		}
		expected := "tmux new-session -d -s new-sess"
		if m.runs[0] != expected {
			t.Errorf("expected %q, got %q", expected, m.runs[0])
		}
	})
}

func TestKillSessionWithMock(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		if err := KillSession("old-sess"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "tmux kill-session -t old-sess"
		if m.runs[0] != expected {
			t.Errorf("expected %q, got %q", expected, m.runs[0])
		}
	})
}
