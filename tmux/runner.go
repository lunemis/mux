package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Output(name string, args ...string) ([]byte, error)
	Run(name string, args ...string) error
}

type execRunner struct{}

func (execRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (execRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// tmux reports the real reason ("duplicate session: dev") on stderr;
		// fold it into the error so the UI shows more than "exit status 1".
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%s (%w)", msg, err)
		}
		return err
	}
	return nil
}

// runner is the package-level command runner, replaceable in tests.
var runner CommandRunner = execRunner{}

// SetRunner replaces the command runner (for testing).
func SetRunner(r CommandRunner) {
	runner = r
}
