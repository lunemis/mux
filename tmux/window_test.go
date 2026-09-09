package tmux

import "testing"

func TestParseWindowLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
		check   func(t *testing.T, w Window)
	}{
		{
			name: "active window",
			line: "0|editor|1",
			check: func(t *testing.T, w Window) {
				if w.Index != 0 {
					t.Errorf("Index = %d, want 0", w.Index)
				}
				if w.Name != "editor" {
					t.Errorf("Name = %q, want editor", w.Name)
				}
				if !w.Active {
					t.Error("Active = false, want true")
				}
			},
		},
		{
			name: "inactive window with spaces in name",
			line: "2|build watch|0",
			check: func(t *testing.T, w Window) {
				if w.Name != "build watch" {
					t.Errorf("Name = %q, want %q", w.Name, "build watch")
				}
				if w.Active {
					t.Error("Active = true, want false")
				}
			},
		},
		{
			name:    "too few fields",
			line:    "0|editor",
			wantErr: true,
		},
		{
			name:    "non-numeric index",
			line:    "abc|editor|1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := parseWindowLine(tt.line)
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
				tt.check(t, w)
			}
		})
	}
}

func TestParsePaneLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
		check   func(t *testing.T, p Pane)
	}{
		{
			name: "active pane",
			line: "0|nvim|1|120|40",
			check: func(t *testing.T, p Pane) {
				if p.Index != 0 {
					t.Errorf("Index = %d, want 0", p.Index)
				}
				if p.Command != "nvim" {
					t.Errorf("Command = %q, want nvim", p.Command)
				}
				if !p.Active {
					t.Error("Active = false, want true")
				}
				if p.Width != 120 {
					t.Errorf("Width = %d, want 120", p.Width)
				}
				if p.Height != 40 {
					t.Errorf("Height = %d, want 40", p.Height)
				}
			},
		},
		{
			name: "inactive pane",
			line: "1|zsh|0|80|24",
			check: func(t *testing.T, p Pane) {
				if p.Active {
					t.Error("Active = true, want false")
				}
			},
		},
		{
			name:    "too few fields",
			line:    "0|nvim|1|120",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parsePaneLine(tt.line)
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
				tt.check(t, p)
			}
		})
	}
}

func TestListWindowsWithMock(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		out := "0|editor|1\n1|server|0\n2|logs|0"
		m.OnOutput([]byte(out), nil, "tmux", "list-windows", "-t", "my-session", "-F", windowListFormat)

		windows, err := ListWindows("my-session")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(windows) != 3 {
			t.Fatalf("expected 3 windows, got %d", len(windows))
		}
		if windows[0].Name != "editor" || !windows[0].Active {
			t.Errorf("windows[0] = %+v, want {editor, active}", windows[0])
		}
		if windows[2].Index != 2 {
			t.Errorf("windows[2].Index = %d, want 2", windows[2].Index)
		}
		for _, w := range windows {
			if w.Panes != nil {
				t.Errorf("window %q has non-nil Panes (should be lazy)", w.Name)
			}
		}
	})
}

func TestListWindowsSortsByIndex(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		// tmux returns out-of-order (unlikely but defensive)
		out := "2|c|0\n0|a|1\n1|b|0"
		m.OnOutput([]byte(out), nil, "tmux", "list-windows", "-t", "s", "-F", windowListFormat)

		windows, err := ListWindows("s")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i, w := range windows {
			if w.Index != i {
				t.Errorf("windows[%d].Index = %d, want %d", i, w.Index, i)
			}
		}
	})
}

func TestListPanesWithMock(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		out := "0|nvim|1|120|40\n1|zsh|0|120|40"
		m.OnOutput([]byte(out), nil, "tmux", "list-panes", "-t", "my-session:2", "-F", paneListFormat)

		panes, err := ListPanes("my-session", 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(panes) != 2 {
			t.Fatalf("expected 2 panes, got %d", len(panes))
		}
		if panes[0].Command != "nvim" || !panes[0].Active {
			t.Errorf("panes[0] = %+v, want {nvim, active}", panes[0])
		}
	})
}

func TestMoveWindowUsesNextFreeDestinationIndex(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		if err := MoveWindow("source", 2, "destination"); err != nil {
			t.Fatalf("MoveWindow() error = %v", err)
		}
		if len(m.runs) != 1 {
			t.Fatalf("run calls = %d, want 1", len(m.runs))
		}
		want := "tmux move-window -d -s source:2 -t destination:"
		if m.runs[0] != want {
			t.Errorf("run = %q, want %q", m.runs[0], want)
		}
	})
}

func TestMoveWindowDoesNotBlockFinalSourceWindow(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		if err := MoveWindow("single-window-source", 0, "destination"); err != nil {
			t.Fatalf("MoveWindow() error = %v", err)
		}
		want := "tmux move-window -d -s single-window-source:0 -t destination:"
		if len(m.runs) != 1 || m.runs[0] != want {
			t.Errorf("run calls = %q, want [%q]", m.runs, want)
		}
	})
}

func TestListWindowsEmpty(t *testing.T) {
	withMock(t, func(m *mockRunner) {
		m.OnOutput([]byte(""), nil, "tmux", "list-windows", "-t", "empty", "-F", windowListFormat)

		windows, err := ListWindows("empty")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(windows) != 0 {
			t.Errorf("expected 0 windows, got %d", len(windows))
		}
	})
}
