package tmux

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

type blockingProcessRunner struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (r *blockingProcessRunner) Output(string, ...string) ([]byte, error) {
	if r.calls.Add(1) == 1 {
		close(r.started)
		<-r.release
	}
	return []byte("1 0\n"), nil
}

func (*blockingProcessRunner) Run(string, ...string) error { return nil }

func TestIsAICommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"claude", true},
		{"codex", true},
		{"aider", true},
		{"gemini", true},
		{"bash", false},
		{"vim", false},
		{"", false},
		{"Claude", false}, // case-sensitive
	}

	for _, tt := range tests {
		if got := IsAICommand(tt.cmd); got != tt.want {
			t.Errorf("IsAICommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestResolveCommandCache(t *testing.T) {
	// Clear cache before test
	cmdCacheMu.Lock()
	cmdCache = make(map[int]cachedCommand)
	cmdCacheMu.Unlock()

	// Pre-populate cache with a known result
	cmdCacheMu.Lock()
	cmdCache[99999] = cachedCommand{
		command:   "claude",
		expiresAt: time.Now().Add(10 * time.Second),
	}
	cmdCacheMu.Unlock()

	// Should return cached value without calling pgrep/ps
	result := resolveCommand(99999, "bash")
	if result != "claude" {
		t.Errorf("expected cached 'claude', got %q", result)
	}

	// Expire the cache entry
	cmdCacheMu.Lock()
	cmdCache[99999] = cachedCommand{
		command:   "claude",
		expiresAt: time.Now().Add(-1 * time.Second),
	}
	cmdCacheMu.Unlock()

	// Expired cache should fall through (pgrep will fail for fake PID, returning rawCmd)
	result = resolveCommand(99999, "bash")
	if result != "bash" {
		t.Errorf("expected fallback 'bash' after cache expiry, got %q", result)
	}
}

func TestResolveCommandDirectAI(t *testing.T) {
	// If rawCmd is already an AI command, should return immediately (no cache needed)
	result := resolveCommand(12345, "claude")
	if result != "claude" {
		t.Errorf("expected 'claude', got %q", result)
	}
}

func TestResolveSessionCommandsBatchesProcessLookup(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		m.OnOutput([]byte("42 100\n43 200\n44 42\n45 300\nmalformed\n"), nil,
			"ps", "-e", "-o", "pid=", "-o", "ppid=")
		m.OnOutput([]byte("42 /usr/bin/bash\n43 /usr/local/bin/codex --quiet\n44 /usr/bin/claude\nmalformed\n"), nil,
			"ps", "-o", "pid=", "-o", "args=", "-p", "42,43")

		sessions := []Session{
			{Name: "shell", ActiveCommand: "bash", PanePID: 100},
			{Name: "agent", ActiveCommand: "zsh", PanePID: 200},
			{Name: "direct", ActiveCommand: "claude", PanePID: 300},
			{Name: "invalid", ActiveCommand: "fish"},
		}
		resolveSessionCommands(sessions)

		if got := sessions[0].ActiveCommand; got != "bash" {
			t.Errorf("shell command = %q, want bash; grandchild AI must not be attributed", got)
		}
		if got := sessions[1].ActiveCommand; got != "codex" {
			t.Errorf("agent command = %q, want codex", got)
		}
		if got := sessions[2].ActiveCommand; got != "claude" {
			t.Errorf("direct command = %q, want claude", got)
		}
		if got := sessions[3].ActiveCommand; got != "fish" {
			t.Errorf("invalid PID command = %q, want fish", got)
		}

		assertOutputCalls(t, m, []string{
			"ps -e -o pid= -o ppid=",
			"ps -o pid= -o args= -p 42,43",
		})
	})
}

func TestResolveSessionCommandsSerializesConcurrentMisses(t *testing.T) {
	old := runner
	r := &blockingProcessRunner{started: make(chan struct{}), release: make(chan struct{})}
	SetRunner(r)
	cmdCacheMu.Lock()
	cmdCache = make(map[int]cachedCommand)
	cmdCacheMu.Unlock()
	defer func() { runner = old }()

	firstDone := make(chan struct{})
	go func() {
		resolveSessionCommands([]Session{{ActiveCommand: "bash", PanePID: 100}})
		close(firstDone)
	}()
	<-r.started

	secondDone := make(chan struct{})
	go func() {
		resolveSessionCommands([]Session{{ActiveCommand: "bash", PanePID: 100}})
		close(secondDone)
	}()

	secondFinishedBeforeFirst := false
	select {
	case <-secondDone:
		secondFinishedBeforeFirst = true
	case <-time.After(20 * time.Millisecond):
	}
	close(r.release)
	<-firstDone
	<-secondDone

	if secondFinishedBeforeFirst {
		t.Error("concurrent miss was not serialized behind the in-flight batch")
	}
	if got := r.calls.Load(); got != 1 {
		t.Errorf("process snapshots = %d, want 1", got)
	}
}

func TestResolveSessionCommandsNoChildrenUsesOnlyTopology(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		m.OnOutput([]byte("1 0\n"), nil,
			"ps", "-e", "-o", "pid=", "-o", "ppid=")
		sessions := []Session{{ActiveCommand: "bash", PanePID: 100}}

		resolveSessionCommands(sessions)

		if got := sessions[0].ActiveCommand; got != "bash" {
			t.Errorf("command = %q, want bash", got)
		}
		assertOutputCalls(t, m, []string{"ps -e -o pid= -o ppid="})
	})
}

func TestResolveSessionCommandsCacheHitStaysOutOfTriggeredBatch(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		cmdCacheMu.Lock()
		cmdCache[100] = cachedCommand{command: "claude", expiresAt: time.Now().Add(time.Minute)}
		cmdCacheMu.Unlock()
		m.OnOutput([]byte("42 100\n43 200\n"), nil,
			"ps", "-e", "-o", "pid=", "-o", "ppid=")
		m.OnOutput([]byte("43 /usr/local/bin/codex\n"), nil,
			"ps", "-o", "pid=", "-o", "args=", "-p", "43")
		sessions := []Session{
			{ActiveCommand: "bash", PanePID: 100},
			{ActiveCommand: "zsh", PanePID: 200},
		}

		resolveSessionCommands(sessions)

		if got := sessions[0].ActiveCommand; got != "claude" {
			t.Errorf("cached command = %q, want claude", got)
		}
		if got := sessions[1].ActiveCommand; got != "codex" {
			t.Errorf("uncached command = %q, want codex", got)
		}
		assertOutputCalls(t, m, []string{
			"ps -e -o pid= -o ppid=",
			"ps -o pid= -o args= -p 43",
		})
	})
}

func TestResolveSessionCommandsFallsBackWhenTopologyLookupFails(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		m.OnOutput(nil, fmt.Errorf("topology ps failed"),
			"ps", "-e", "-o", "pid=", "-o", "ppid=")
		m.OnOutput([]byte("42\n"), nil, "pgrep", "-P", "100")
		m.OnOutput([]byte("/usr/local/bin/claude --quiet\n"), nil,
			"ps", "-o", "args=", "-p", "42")
		sessions := []Session{{ActiveCommand: "bash", PanePID: 100}}

		resolveSessionCommands(sessions)

		if got := sessions[0].ActiveCommand; got != "claude" {
			t.Errorf("fallback command = %q, want claude", got)
		}
		assertOutputCalls(t, m, []string{
			"ps -e -o pid= -o ppid=",
			"pgrep -P 100",
			"ps -o args= -p 42",
		})
	})
}

func TestResolveSessionCommandsFallsBackWhenTargetedLookupFails(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		m.OnOutput([]byte("42 100\n"), nil,
			"ps", "-e", "-o", "pid=", "-o", "ppid=")
		m.OnOutput(nil, fmt.Errorf("targeted ps failed"),
			"ps", "-o", "pid=", "-o", "args=", "-p", "42")
		m.OnOutput([]byte("42\n"), nil, "pgrep", "-P", "100")
		m.OnOutput([]byte("/usr/local/bin/claude --quiet\n"), nil,
			"ps", "-o", "args=", "-p", "42")
		sessions := []Session{{ActiveCommand: "bash", PanePID: 100}}

		resolveSessionCommands(sessions)

		if got := sessions[0].ActiveCommand; got != "claude" {
			t.Errorf("fallback command = %q, want claude", got)
		}
		assertOutputCalls(t, m, []string{
			"ps -e -o pid= -o ppid=",
			"ps -o pid= -o args= -p 42",
			"pgrep -P 100",
			"ps -o args= -p 42",
		})
	})
}

func assertOutputCalls(t *testing.T, m *mockRunner, want []string) {
	t.Helper()
	if len(m.outputCalls) != len(want) {
		t.Fatalf("process calls = %q, want %q", m.outputCalls, want)
	}
	for i := range want {
		if m.outputCalls[i] != want[i] {
			t.Errorf("process call %d = %q, want %q", i, m.outputCalls[i], want[i])
		}
	}
}
