package tmux

import (
	"fmt"
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
			line: "$0|my-session|2|1711900000|1|/home/user/project|1711900100|bash|12345",
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
			},
		},
		{
			name: "not attached",
			line: "$1|dev|1|1711900000|0|/tmp|1711900050|zsh|99999",
			check: func(t *testing.T, s Session) {
				if s.Attached {
					t.Error("Attached = true, want false")
				}
			},
		},
		{
			name:    "too few fields",
			line:    "bad|line|only",
			wantErr: true,
		},
		{
			// A pipe inside a free-form field (name, path) shifts later
			// columns — a known parsing limitation — but the session ID is
			// emitted first and can never contain the delimiter, so targeting
			// (kill/rename/attach) stays correct even for such sessions.
			name: "pipe in session name keeps the session ID intact",
			line: "$2|my|piped|1|" + itoa(now) + "|0|/home/user|" + itoa(now) + "|bash|123",
			check: func(t *testing.T, s Session) {
				if s.ID != "$2" {
					t.Errorf("ID = %q, want %q", s.ID, "$2")
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

func itoa(n int64) string {
	return fmt.Sprintf("%d", n)
}

// #{session_id}는 이름과 달리 특수문자(. : $ =)가 있을 수 없고 rename에도
// 불변이므로 모든 -t 타깃의 기준이 된다.
func TestParseLineSessionID(t *testing.T) {
	line := "$3|my.session|2|1711900000|1|/home/user/project|1711900100|bash|12345"
	s, err := parseLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != "$3" {
		t.Errorf("ID = %q, want %q", s.ID, "$3")
	}
	if s.Name != "my.session" {
		t.Errorf("Name = %q, want %q", s.Name, "my.session")
	}
	if s.WindowCount != 2 {
		t.Errorf("WindowCount = %d, want 2", s.WindowCount)
	}
	if s.PanePID != 12345 {
		t.Errorf("PanePID = %d, want 12345", s.PanePID)
	}
}
