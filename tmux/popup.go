package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	popupWidth     = "100%"
	popupHeight    = "100%"
	minTmuxVersion = 3.2
	DefaultBindKey = "m"

	// keybindMarker tags lines we own so we can replace them idempotently.
	keybindMarker = "# tmux-peeker popup keybinding"

	// ohMyTmuxSignature is the first line of gpakosz/.tmux's bundled .tmux.conf.
	// It opens a heredoc that lets the file double as a shell script when
	// processed via `cut -c3- | sh -s ...`.
	ohMyTmuxSignature = "# : << 'EOF'"

	// ohMyTmuxSentinel marks the end of user-editable territory in
	// .tmux.conf.local. oh-my-tmux explicitly warns against writing past it.
	ohMyTmuxSentinel = `# "$@"`
)

// OpenPopup opens tmux-peeker inside a tmux display-popup overlay and forwards
// args to the current process. It must be called from inside a tmux session.
func OpenPopup(args ...string) error {
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("tmux-peeker popup must be run inside a tmux session")
	}

	version, err := getTmuxVersion()
	if err != nil {
		return fmt.Errorf("failed to detect tmux version: %w", err)
	}
	if version < minTmuxVersion {
		return fmt.Errorf("tmux %.1f+ required for popup (current: %.1f)", minTmuxVersion, version)
	}

	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find tmux-peeker executable: %w", err)
	}

	origin := currentSessionName()
	cmd := exec.Command("tmux", popupCommandArgs(executablePath, origin, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func popupCommandArgs(executablePath, origin string, args ...string) []string {
	popupArgs := []string{
		"display-popup",
		"-B",
		"-E",
		"-w", popupWidth,
		"-h", popupHeight,
	}
	if origin != "" {
		popupArgs = append(popupArgs, "-e", originSessionEnv+"="+origin)
	}
	popupArgs = append(popupArgs, executablePath)
	return append(popupArgs, args...)
}

func popupBindLine(key, executablePath string) string {
	return fmt.Sprintf(`bind %s run-shell '%s=#{q:session_name} "%s" popup'`,
		key, originSessionEnv, executablePath)
}

func findTmuxConf() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %w", err)
	}

	// Candidate paths in priority order:
	//   $XDG_CONFIG_HOME/tmux/tmux.conf → ~/.config/tmux/tmux.conf → ~/.tmux.conf
	var candidates []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "tmux", "tmux.conf"))
	}
	candidates = append(candidates,
		filepath.Join(home, ".config", "tmux", "tmux.conf"),
		filepath.Join(home, ".tmux.conf"),
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return filepath.Join(home, ".tmux.conf"), nil
}

// findTmuxConfLocal returns the path where oh-my-tmux user customizations live,
// derived from the matched tmux.conf path so XDG and home variants stay paired.
func findTmuxConfLocal(confPath string) string {
	dir := filepath.Dir(confPath)
	base := filepath.Base(confPath)
	switch base {
	case "tmux.conf":
		return filepath.Join(dir, "tmux.conf.local")
	case ".tmux.conf":
		return filepath.Join(dir, ".tmux.conf.local")
	default:
		return confPath + ".local"
	}
}

// isOhMyTmux detects gpakosz/.tmux installs using a hybrid of two signals:
//
//   - Strategy A (symlink): confPath is a symlink whose target lives under a
//     `.tmux/` directory — matches the upstream installer's layout.
//   - Strategy B (signature): confPath's first line is `# : << 'EOF'` — the
//     heredoc opener oh-my-tmux uses to make the conf file double as a shell
//     script. Catches users who copied files instead of symlinking.
//
// Either signal alone is sufficient.
func isOhMyTmux(confPath string) bool {
	if info, err := os.Lstat(confPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(confPath); err == nil {
			// Resolve relative symlinks against the symlink's directory.
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(confPath), target)
			}
			if strings.Contains(filepath.ToSlash(target), "/.tmux/") {
				return true
			}
		}
	}

	f, err := os.Open(confPath)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, len(ohMyTmuxSignature))
	n, _ := f.Read(buf)
	return strings.TrimSpace(string(buf[:n])) == ohMyTmuxSignature
}

// SetupKeybind adds a popup keybinding to the user's tmux config file.
//
// For oh-my-tmux installs the bind line is routed to .tmux.conf.local and
// inserted before the `# "$@"` sentinel (oh-my-tmux marks everything below
// that line as off-limits). Existing bindings owned by other tools are left
// untouched.
func SetupKeybind(key string) error {
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find tmux-peeker executable: %w", err)
	}

	confPath, err := findTmuxConf()
	if err != nil {
		return err
	}
	bindLine := popupBindLine(key, executablePath)

	if isOhMyTmux(confPath) {
		localPath := findTmuxConfLocal(confPath)
		if err := writeBindToLocal(localPath, bindLine); err != nil {
			return err
		}
		fmt.Printf("Detected oh-my-tmux. Added to %s:\n  %s\n\n", localPath, bindLine)
		fmt.Printf("Reload tmux config:\n  tmux source-file %s\n\n", localPath)
		fmt.Printf("Then press: prefix + %s (default prefix: Ctrl+b)\n", key)
		return nil
	}

	if err := upsertBindLine(confPath, bindLine, true); err != nil {
		return err
	}
	fmt.Printf("Added to %s:\n  %s\n\n", confPath, bindLine)
	fmt.Printf("Reload tmux config:\n  tmux source-file %s\n\n", confPath)
	fmt.Printf("Then press: prefix + %s (default prefix: Ctrl+b)\n", key)
	return nil
}

// upsertBindLine writes bindLine into path. If a line tagged with
// keybindMarker already exists it is replaced in place; otherwise the line
// is appended (createIfMissing controls whether a missing file is allowed).
func upsertBindLine(path, bindLine string, createIfMissing bool) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		if !createIfMissing {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
	}

	tagged := bindLine + "  " + keybindMarker
	lines := strings.Split(string(content), "\n")
	replaced := false
	for i, line := range lines {
		if strings.Contains(line, keybindMarker) {
			lines[i] = tagged
			replaced = true
		}
	}
	if !replaced {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, tagged)
	}

	result := strings.Join(lines, "\n")
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	if err := os.WriteFile(path, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// writeBindToLocal upserts bindLine into an oh-my-tmux .tmux.conf.local,
// inserting before the `# "$@"` sentinel. Falls back to append if the sentinel
// is missing (user may have stripped it). Creates the file if absent.
func writeBindToLocal(path, bindLine string) error {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	tagged := bindLine + "  " + keybindMarker
	lines := strings.Split(string(content), "\n")

	// Replace existing marker line in place.
	for i, line := range lines {
		if strings.Contains(line, keybindMarker) {
			lines[i] = tagged
			result := strings.Join(lines, "\n")
			if !strings.HasSuffix(result, "\n") {
				result += "\n"
			}
			return os.WriteFile(path, []byte(result), 0644)
		}
	}

	// Insert before the sentinel if present.
	sentinelIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == ohMyTmuxSentinel {
			sentinelIdx = i
			break
		}
	}

	if sentinelIdx >= 0 {
		insertion := []string{"", tagged, ""}
		newLines := make([]string, 0, len(lines)+len(insertion))
		newLines = append(newLines, lines[:sentinelIdx]...)
		newLines = append(newLines, insertion...)
		newLines = append(newLines, lines[sentinelIdx:]...)
		lines = newLines
	} else {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, tagged)
	}

	result := strings.Join(lines, "\n")
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	if err := os.WriteFile(path, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func getTmuxVersion() (float64, error) {
	out, err := exec.Command("tmux", "-V").Output()
	if err != nil {
		return 0, err
	}
	// Output: "tmux 3.4" or "tmux 3.2a"
	s := strings.TrimSpace(string(out))
	s = strings.TrimPrefix(s, "tmux ")
	// Strip trailing letter (e.g. "3.2a" -> "3.2")
	var version float64
	fmt.Sscanf(s, "%f", &version)
	return version, nil
}
