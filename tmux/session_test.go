package tmux

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseLine(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name    string
		line    string
		wantErr bool
		check   func(t *testing.T, s Session)
	}{
		{
			name: "valid line",
			line: strings.Join([]string{
				"my-session", "2", "1711900000", "1", "/home/user/project", "1711900100", "nvim",
			}, sessionFieldSeparator),
			check: func(t *testing.T, s Session) {
				if s.Name != "my-session" {
					t.Errorf("Name = %q, want %q", s.Name, "my-session")
				}
				if s.WindowCount != 2 {
					t.Errorf("WindowCount = %d, want %d", s.WindowCount, 2)
				}
				if !s.Attached {
					t.Error("Attached = false, want true")
				}
				if s.Directory != "/home/user/project" {
					t.Errorf("Directory = %q, want %q", s.Directory, "/home/user/project")
				}
				if got := s.LastAttached.Unix(); got != 1711900100 {
					t.Errorf("LastAttached = %d, want 1711900100", got)
				}
				if s.ActiveCommand != "nvim" {
					t.Errorf("ActiveCommand = %q, want nvim", s.ActiveCommand)
				}
			},
		},
		{
			name: "not attached",
			line: strings.Join([]string{
				"dev", "1", "1711900000", "0", "/tmp", "0", "zsh",
			}, sessionFieldSeparator),
			check: func(t *testing.T, s Session) {
				if s.Attached {
					t.Error("Attached = true, want false")
				}
				if !s.LastAttached.IsZero() {
					t.Errorf("LastAttached = %v, want zero", s.LastAttached)
				}
			},
		},
		{
			name:    "too few fields",
			line:    "bad|line|only",
			wantErr: true,
		},
		{
			name: "session with pipe in path still works with SplitN",
			line: strings.Join([]string{
				"test", "1", itoa(now), "0", "/home/user|project", itoa(now), "bash",
			}, sessionFieldSeparator),
			check: func(t *testing.T, s Session) {
				if s.Name != "test" {
					t.Errorf("Name = %q, want %q", s.Name, "test")
				}
				if s.Directory != "/home/user|project" || s.ActiveCommand != "bash" {
					t.Errorf("parsed path/command = %q/%q, want pipe path and bash", s.Directory, s.ActiveCommand)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := parseLine(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}

func TestCurrentSessionForSwitcher(t *testing.T) {
	sessions := []Session{{Name: "attached", Attached: true}, {Name: "reported", Attached: true}}
	if got := currentSessionForSwitcher(sessions, "reported"); got != "reported" {
		t.Errorf("valid reported session = %q, want reported", got)
	}

	sessions[1].Attached = false
	if got := currentSessionForSwitcher(sessions, "popup-pane"); got != "attached" {
		t.Errorf("invalid reported session fallback = %q, want attached", got)
	}
}

func TestSoleAttachedSession(t *testing.T) {
	tests := []struct {
		name     string
		sessions []Session
		want     string
	}{
		{name: "one attached", sessions: []Session{{Name: "current", Attached: true}, {Name: "other"}}, want: "current"},
		{name: "none attached", sessions: []Session{{Name: "one"}, {Name: "two"}}},
		{name: "multiple attached is ambiguous", sessions: []Session{{Name: "one", Attached: true}, {Name: "two", Attached: true}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := soleAttachedSession(test.sessions); got != test.want {
				t.Errorf("soleAttachedSession() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSortSessionsForSwitcher(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	sessions := []Session{
		{Name: "attached-old", Attached: true, LastAttached: base.Add(-2 * time.Hour), Created: base.Add(-24 * time.Hour)},
		{Name: "recent", Attached: false, LastAttached: base.Add(-time.Minute), Created: base.Add(-48 * time.Hour)},
		{Name: "never-old", Created: base.Add(-72 * time.Hour)},
		{Name: "never-new", Created: base.Add(-time.Hour)},
		{Name: "zeta", LastAttached: base.Add(-time.Hour), Created: base},
		{Name: "alpha", LastAttached: base.Add(-time.Hour), Created: base},
	}

	sortSessionsForSwitcher(sessions, "recent")

	got := make([]string, len(sessions))
	for i, session := range sessions {
		got[i] = session.Name
	}
	want := []string{"recent", "alpha", "zeta", "attached-old", "never-new", "never-old"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted names = %q, want %q", got, want)
		}
		if sessions[i].Current != (i == 0) {
			t.Errorf("session %q Current = %t, want %t", sessions[i].Name, sessions[i].Current, i == 0)
		}
	}
}

func TestSortSessionsForSwitcherPinsCurrentAheadOfNewerSession(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	sessions := []Session{
		{Name: "newest", LastAttached: base},
		{Name: "current", LastAttached: base.Add(-time.Hour)},
		{Name: "older", LastAttached: base.Add(-2 * time.Hour)},
	}

	sortSessionsForSwitcher(sessions, "current")
	got := []string{sessions[0].Name, sessions[1].Name, sessions[2].Name}
	want := []string{"current", "newest", "older"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted names = %q, want %q", got, want)
		}
	}
	if !sessions[0].Current {
		t.Error("pinned current session is not marked Current")
	}
}

func TestSortSessionsForSwitcherWithoutCurrentSessionKeepsMRUFirst(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	sessions := []Session{
		{Name: "older", LastAttached: base.Add(-time.Hour)},
		{Name: "recent", LastAttached: base.Add(-time.Minute)},
	}

	sortSessionsForSwitcher(sessions, "")
	if sessions[0].Name != "recent" || sessions[1].Name != "older" {
		t.Errorf("sorted names = [%s %s], want [recent older]", sessions[0].Name, sessions[1].Name)
	}
}

func itoa(n int64) string {
	return fmt.Sprintf("%d", n)
}
