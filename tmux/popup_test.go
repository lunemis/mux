package tmux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPopupCommandArgsCarryResolvedOriginSession(t *testing.T) {
	got := strings.Join(popupCommandArgs("/bin/tmux-peeker", "work", "--theme", "default"), " ")
	want := "display-popup -B -E -w 100% -h 100% -e TMUX_PEEKER_ORIGIN_SESSION=work /bin/tmux-peeker --theme default"
	if got != want {
		t.Errorf("popup args = %q, want %q", got, want)
	}
}

func TestPopupBindLineExpandsOriginBeforeLaunchingPopup(t *testing.T) {
	got := popupBindLine("m", "/bin/tmux-peeker")
	want := `bind m run-shell 'TMUX_PEEKER_ORIGIN_SESSION=#{q:session_name} "/bin/tmux-peeker" popup'`
	if got != want {
		t.Errorf("popup bind line = %q, want %q", got, want)
	}
}

func TestOpenPopupPassesResolvedOriginToChild(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args")
	fakeTmux := filepath.Join(dir, "tmux")
	script := `#!/bin/sh
if [ "$1" = "-V" ]; then
	printf 'tmux 3.4\n'
	exit 0
fi
printf '%s\n' "$@" > "$TMUX_PEEKER_TEST_ARGS"
`
	if err := os.WriteFile(fakeTmux, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TMUX", filepath.Join(dir, "socket")+",1,0")
	t.Setenv(originSessionEnv, "work")
	t.Setenv("TMUX_PEEKER_TEST_ARGS", argsPath)

	if err := OpenPopup("--theme", "default"); err != nil {
		t.Fatalf("OpenPopup() error = %v", err)
	}
	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	executablePath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{
		"display-popup", "-B", "-E", "-w", popupWidth, "-h", popupHeight,
		"-e", originSessionEnv + "=work", executablePath, "--theme", "default",
	}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("popup process args = %q, want %q", got, want)
	}
}

func TestInstallerPopupBindCarriesOriginSession(t *testing.T) {
	data, err := os.ReadFile("../install.sh")
	if err != nil {
		t.Fatal(err)
	}
	want := `local line="bind-key m run-shell 'TMUX_PEEKER_ORIGIN_SESSION=#{q:session_name} \"tmux-peeker\" popup'"`
	if !strings.Contains(string(data), want) {
		t.Errorf("install.sh does not contain popup binding %q", want)
	}
}

const sampleOhMyTmuxLocal = `# -- general -------------------------------------------------------------------

tmux_conf_24b_colour=true

# -- custom variables ----------------------------------------------------------

# EOF

# "$@"
`

func TestFindTmuxConfLocal(t *testing.T) {
	tests := []struct {
		conf string
		want string
	}{
		{conf: "/home/u/.tmux.conf", want: "/home/u/.tmux.conf.local"},
		{conf: "/home/u/.config/tmux/tmux.conf", want: "/home/u/.config/tmux/tmux.conf.local"},
		{conf: "/etc/odd/path", want: "/etc/odd/path.local"},
	}
	for _, tt := range tests {
		if got := findTmuxConfLocal(tt.conf); got != tt.want {
			t.Errorf("findTmuxConfLocal(%q) = %q, want %q", tt.conf, got, tt.want)
		}
	}
}

func TestIsOhMyTmux_Signature(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := os.WriteFile(path, []byte(ohMyTmuxSignature+"\n# rest of file\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !isOhMyTmux(path) {
		t.Error("expected oh-my-tmux to be detected via signature")
	}
}

func TestIsOhMyTmux_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".tmux", ".tmux.conf")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("# anything\n"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".tmux.conf")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if !isOhMyTmux(link) {
		t.Error("expected oh-my-tmux to be detected via symlink target")
	}
}

func TestIsOhMyTmux_PlainConf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := os.WriteFile(path, []byte("set -g mouse on\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if isOhMyTmux(path) {
		t.Error("plain config should not be detected as oh-my-tmux")
	}
}

func TestIsOhMyTmux_Missing(t *testing.T) {
	if isOhMyTmux(filepath.Join(t.TempDir(), "does-not-exist")) {
		t.Error("missing file should not be detected as oh-my-tmux")
	}
}

func TestUpsertBindLine_AppendsToPlainConf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := os.WriteFile(path, []byte("set -g mouse on\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := upsertBindLine(path, `bind m display-popup -E "/bin/tmux-peeker"`, true); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "set -g mouse on") {
		t.Error("existing config dropped")
	}
	if !strings.Contains(string(got), keybindMarker) {
		t.Error("marker not written")
	}
}

func TestUpsertBindLine_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf")
	if err := upsertBindLine(path, `bind m display-popup -E "/bin/old"`, true); err != nil {
		t.Fatal(err)
	}
	if err := upsertBindLine(path, `bind m display-popup -E "/bin/new"`, true); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "/bin/old") {
		t.Error("old bind should have been replaced")
	}
	if !strings.Contains(string(got), "/bin/new") {
		t.Error("new bind missing")
	}
	if n := strings.Count(string(got), keybindMarker); n != 1 {
		t.Errorf("expected exactly one marker line, got %d", n)
	}
}

func TestWriteBindToLocal_InsertsBeforeSentinel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf.local")
	if err := os.WriteFile(path, []byte(sampleOhMyTmuxLocal), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/tmux-peeker"`); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)

	bindIdx := strings.Index(s, "/bin/tmux-peeker")
	sentIdx := strings.Index(s, ohMyTmuxSentinel)
	if bindIdx == -1 {
		t.Fatal("bind line missing")
	}
	if sentIdx == -1 {
		t.Fatal("sentinel was removed")
	}
	if bindIdx >= sentIdx {
		t.Errorf("bind line should appear before sentinel (bind=%d, sentinel=%d)", bindIdx, sentIdx)
	}
	// The cut -c3- shell extraction would treat any non-`# `-prefixed line as
	// shell input. Sanity check: the tagged line still starts with `bind`,
	// not a stripped `nd m display-popup …`.
	if !strings.Contains(s, "bind m display-popup") {
		t.Error("bind line should not have been mangled")
	}
}

func TestWriteBindToLocal_NoSentinel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf.local")
	if err := os.WriteFile(path, []byte("set -g mouse on\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/tmux-peeker"`); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "/bin/tmux-peeker") {
		t.Error("bind line missing when sentinel absent")
	}
}

func TestWriteBindToLocal_CreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf.local")
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/tmux-peeker"`); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should have been created: %v", err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "/bin/tmux-peeker") {
		t.Error("bind line missing in newly created file")
	}
}

func TestWriteBindToLocal_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tmux.conf.local")
	if err := os.WriteFile(path, []byte(sampleOhMyTmuxLocal), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/old"`); err != nil {
		t.Fatal(err)
	}
	if err := writeBindToLocal(path, `bind m display-popup -E "/bin/new"`); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if strings.Contains(s, "/bin/old") {
		t.Error("old bind should have been replaced on second write")
	}
	if n := strings.Count(s, keybindMarker); n != 1 {
		t.Errorf("expected exactly one marker line after re-run, got %d", n)
	}
	// Sentinel must still be the last meaningful line.
	if !strings.Contains(s, ohMyTmuxSentinel) {
		t.Error("sentinel lost on rewrite")
	}
}

func TestUpsertBindLinePreservesUpstreamMuxBinding(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tmux.conf")
	upstream := `bind m display-popup -E "/usr/local/bin/mux"  # mux popup keybinding`
	if err := os.WriteFile(path, []byte(upstream+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	bind := `bind m run-shell 'TMUX_PEEKER_ORIGIN_SESSION=#{q:session_name} "/usr/local/bin/tmux-peeker" popup'`
	if err := upsertBindLine(path, bind, true); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got)
	if !strings.Contains(content, upstream) {
		t.Error("upstream mux binding was modified")
	}
	if !strings.Contains(content, "# tmux-peeker popup keybinding") {
		t.Error("tmux-peeker binding marker is missing")
	}
}
