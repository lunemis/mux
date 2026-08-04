package tmux

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockRunner records calls and returns pre-configured responses.
type mockRunner struct {
	outputs map[string]mockResult
	runs    []string
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

// tmux writes the actual failure reason ("duplicate session: dev",
// "can't find session: x") to stderr; a bare "exit status 1" is useless to
// the user, so Run must surface it.
func TestExecRunnerRunIncludesStderr(t *testing.T) {
	err := execRunner{}.Run("sh", "-c", "echo 'duplicate session: dev' >&2; exit 1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "duplicate session: dev") {
		t.Errorf("error %q should contain the stderr text", err.Error())
	}
}

func TestListSessionsWithMock(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		now := time.Now().Unix()
		line1 := fmt.Sprintf("$0|dev|2|%d|1|/home/user/dev|%d|bash|100", now-3600, now-60)
		line2 := fmt.Sprintf("$1|ai|1|%d|0|/home/user/ai|%d|claude|200", now-7200, now-120)
		out := line1 + "\n" + line2

		// Mock the list-sessions call
		m.OnOutput([]byte(out), nil, "tmux", "list-sessions", "-F", listFormat)
		// Mock resolveCommand calls — pgrep returns nothing (so rawCmd is used)
		m.OnOutput(nil, fmt.Errorf("no children"), "pgrep", "-P", "100")
		m.OnOutput(nil, fmt.Errorf("no children"), "pgrep", "-P", "200")

		sessions, err := ListSessions()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(sessions))
		}
		// Attached sessions first
		if sessions[0].Name != "dev" {
			t.Errorf("expected first session 'dev', got %q", sessions[0].Name)
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
		if content != "hello world" {
			t.Errorf("expected 'hello world', got %q", content)
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
