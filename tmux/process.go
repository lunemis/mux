package tmux

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const commandCacheTTL = 5 * time.Second

type cachedCommand struct {
	command   string
	expiresAt time.Time
}

var (
	cmdCache     = make(map[int]cachedCommand)
	cmdCacheMu   sync.Mutex
	cmdResolveMu sync.Mutex
)

// resolveCommand returns the logical command name for a pane.
// Results are cached by panePID with a TTL to avoid repeated pgrep/ps calls.
func resolveCommand(panePID int, rawCmd string) string {
	if panePID <= 0 {
		return rawCmd
	}

	if IsAICommand(rawCmd) {
		return rawCmd
	}

	cmdCacheMu.Lock()
	if cached, ok := cmdCache[panePID]; ok && time.Now().Before(cached.expiresAt) {
		cmdCacheMu.Unlock()
		return cached.command
	}
	cmdCacheMu.Unlock()

	result := scanChildProcesses(panePID, rawCmd)
	cacheCommand(panePID, result)
	return result
}

// resolveSessionCommands resolves pane commands in two bounded process queries:
// one process-topology snapshot and one argument lookup restricted to direct
// children of the unresolved panes. The legacy per-pane lookup remains the
// fallback when either process query fails.
func resolveSessionCommands(sessions []Session) {
	cmdResolveMu.Lock()
	defer cmdResolveMu.Unlock()

	now := time.Now()
	unresolved := make([]int, 0, len(sessions))
	parents := make(map[int]struct{}, len(sessions))

	cmdCacheMu.Lock()
	for i := range sessions {
		pid := sessions[i].PanePID
		if pid <= 0 || IsAICommand(sessions[i].ActiveCommand) {
			continue
		}
		if cached, ok := cmdCache[pid]; ok && now.Before(cached.expiresAt) {
			sessions[i].ActiveCommand = cached.command
			continue
		}
		unresolved = append(unresolved, i)
		parents[pid] = struct{}{}
	}
	cmdCacheMu.Unlock()

	if len(unresolved) == 0 {
		return
	}

	out, err := runner.Output("ps", "-e", "-o", "pid=", "-o", "ppid=")
	if err != nil {
		resolveSessionsIndividually(sessions, unresolved)
		return
	}
	childrenByParent, valid := parseProcessTopology(out, parents)
	if !valid {
		resolveSessionsIndividually(sessions, unresolved)
		return
	}

	childSet := make(map[int]struct{})
	for _, children := range childrenByParent {
		for _, pid := range children {
			childSet[pid] = struct{}{}
		}
	}
	childPIDs := make([]int, 0, len(childSet))
	for pid := range childSet {
		childPIDs = append(childPIDs, pid)
	}
	sort.Ints(childPIDs)

	if len(childPIDs) == 0 {
		cacheSessionCommands(sessions, unresolved)
		return
	}

	pidArgs := make([]string, len(childPIDs))
	for i, pid := range childPIDs {
		pidArgs[i] = strconv.Itoa(pid)
	}
	out, err = runner.Output("ps", "-o", "pid=", "-o", "args=", "-p", strings.Join(pidArgs, ","))
	if err != nil {
		for _, i := range unresolved {
			if len(childrenByParent[sessions[i].PanePID]) > 0 {
				sessions[i].ActiveCommand = resolveCommand(sessions[i].PanePID, sessions[i].ActiveCommand)
			}
		}
		cacheSessionCommands(sessions, unresolved)
		return
	}

	argsByPID := parseProcessArgs(out)
	for _, i := range unresolved {
		sessions[i].ActiveCommand = detectAIChild(
			childrenByParent[sessions[i].PanePID], argsByPID, sessions[i].ActiveCommand)
	}
	cacheSessionCommands(sessions, unresolved)
}

func parseProcessTopology(out []byte, parents map[int]struct{}) (map[int][]int, bool) {
	children := make(map[int][]int)
	valid := false
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, pidErr := strconv.Atoi(fields[0])
		ppid, ppidErr := strconv.Atoi(fields[1])
		if pidErr != nil || ppidErr != nil {
			continue
		}
		valid = true
		if _, ok := parents[ppid]; ok {
			children[ppid] = append(children[ppid], pid)
		}
	}
	for ppid := range children {
		sort.Ints(children[ppid])
	}
	return children, valid
}

func parseProcessArgs(out []byte) map[int][]string {
	argsByPID := make(map[int][]string)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err == nil {
			argsByPID[pid] = fields[1:]
		}
	}
	return argsByPID
}

func detectAIChild(childPIDs []int, argsByPID map[int][]string, rawCmd string) string {
	for _, pid := range childPIDs {
		for _, part := range argsByPID[pid] {
			base := filepath.Base(part)
			if IsAICommand(base) {
				return base
			}
		}
	}
	return rawCmd
}

func resolveSessionsIndividually(sessions []Session, indices []int) {
	for _, i := range indices {
		sessions[i].ActiveCommand = resolveCommand(sessions[i].PanePID, sessions[i].ActiveCommand)
	}
}

func cacheSessionCommands(sessions []Session, indices []int) {
	cmdCacheMu.Lock()
	defer cmdCacheMu.Unlock()
	expiresAt := time.Now().Add(commandCacheTTL)
	for _, i := range indices {
		cmdCache[sessions[i].PanePID] = cachedCommand{
			command:   sessions[i].ActiveCommand,
			expiresAt: expiresAt,
		}
	}
}

func cacheCommand(panePID int, command string) {
	cmdCacheMu.Lock()
	cmdCache[panePID] = cachedCommand{
		command:   command,
		expiresAt: time.Now().Add(commandCacheTTL),
	}
	cmdCacheMu.Unlock()
}

// scanChildProcesses inspects child processes of the pane shell to detect AI CLIs.
// Works on both Linux and macOS using pgrep/ps.
func scanChildProcesses(panePID int, rawCmd string) string {
	childPIDs := findChildPIDs(panePID)
	if len(childPIDs) == 0 {
		return rawCmd
	}

	for _, pidStr := range childPIDs {
		args, err := runner.Output("ps", "-o", "args=", "-p", pidStr)
		if err != nil {
			continue
		}
		for _, part := range strings.Fields(string(args)) {
			base := filepath.Base(part)
			if IsAICommand(base) {
				return base
			}
		}
	}
	return rawCmd
}

// findChildPIDs returns child PIDs of the given parent.
// Tries pgrep first, falls back to ps -eo pid,ppid for macOS compatibility.
func findChildPIDs(parentPID int) []string {
	pidStr := fmt.Sprintf("%d", parentPID)

	// Try pgrep first (works reliably on Linux)
	out, err := runner.Output("pgrep", "-P", pidStr)
	if err == nil {
		if fields := strings.Fields(string(out)); len(fields) > 0 {
			return fields
		}
	}

	// Fallback: ps -eo pid,ppid (more reliable on macOS)
	out, err = runner.Output("ps", "-eo", "pid,ppid")
	if err != nil {
		return nil
	}

	var children []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == pidStr {
			children = append(children, fields[0])
		}
	}
	return children
}
