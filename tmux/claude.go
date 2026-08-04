package tmux

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	claudeSessionsTTL = 10 * time.Second
	claudeDir         = ".claude"
	sessionsDir       = "sessions"
	projectsDir       = "projects"

	// maxScanTokenSize caps a single JSONL line; real session logs carry
	// multi-megabyte tool results, so this must stay generous.
	maxScanTokenSize = 16 * 1024 * 1024
)

// TokenUsage holds aggregated token counts and estimated cost for a Claude session.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	CacheRead    int
	CacheWrite   int
	TotalCost    float64 // estimated USD
}

// claudeSessionFile represents the JSON structure of ~/.claude/sessions/{PID}.json.
type claudeSessionFile struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
}

// jsonlMessage is a minimal representation of a JSONL line with usage data.
type jsonlMessage struct {
	Type    string `json:"type"`
	Message struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

type cachedUsage struct {
	usage     *TokenUsage
	expiresAt time.Time
}

var (
	usageCache   = make(map[string]cachedUsage) // sessionID → cached usage
	usageCacheMu sync.Mutex
)

// FindClaudeSession locates a Claude Code session file for a given tmux pane PID.
// It scans child processes to find the Claude PID, then reads its session file.
func FindClaudeSession(panePID int) (sessionID string, cwd string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	// Get child PIDs of the pane shell
	out, err := runner.Output("pgrep", "-P", fmt.Sprintf("%d", panePID))
	if err != nil {
		return "", "", fmt.Errorf("no child processes for pane %d", panePID)
	}

	sessDir := filepath.Join(home, claudeDir, sessionsDir)

	for _, pidStr := range strings.Fields(string(out)) {
		path := filepath.Join(sessDir, pidStr+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var sf claudeSessionFile
		if err := json.Unmarshal(data, &sf); err != nil {
			continue
		}
		return sf.SessionID, sf.CWD, nil
	}

	return "", "", fmt.Errorf("no claude session found for pane %d", panePID)
}

// LoadTokenUsage reads and aggregates token usage from a Claude session's JSONL log.
// Results are cached with a TTL to avoid re-reading large files on every tick.
func LoadTokenUsage(sessionID, cwd string) (*TokenUsage, error) {
	usageCacheMu.Lock()
	if cached, ok := usageCache[sessionID]; ok && time.Now().Before(cached.expiresAt) {
		usageCacheMu.Unlock()
		return cached.usage, nil
	}
	usageCacheMu.Unlock()

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	encoded := encodePath(cwd)
	jsonlPath := filepath.Join(home, claudeDir, projectsDir, encoded, sessionID+".jsonl")

	usage, err := parseTokenUsage(jsonlPath)
	if err != nil {
		return nil, err
	}

	usageCacheMu.Lock()
	usageCache[sessionID] = cachedUsage{
		usage:     usage,
		expiresAt: time.Now().Add(claudeSessionsTTL),
	}
	usageCacheMu.Unlock()

	return usage, nil
}

func parseTokenUsage(path string) (*TokenUsage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	usage := &TokenUsage{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Quick filter: skip lines without "usage"
		if !containsBytes(line, []byte(`"usage"`)) {
			continue
		}

		var msg jsonlMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if msg.Type != "assistant" {
			continue
		}

		u := msg.Message.Usage
		usage.InputTokens += u.InputTokens
		usage.OutputTokens += u.OutputTokens
		usage.CacheRead += u.CacheReadInputTokens
		usage.CacheWrite += u.CacheCreationInputTokens
	}

	if err := scanner.Err(); err != nil {
		// A partial aggregate must not be reported (or cached) as complete.
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	usage.TotalCost = estimateCost(usage)
	return usage, nil
}

// estimateCost calculates an approximate USD cost from token counts.
// Uses Claude Opus 4.6 pricing as default.
func estimateCost(u *TokenUsage) float64 {
	const (
		inputPer1M      = 15.0
		outputPer1M     = 75.0
		cacheReadPer1M  = 1.5
		cacheWritePer1M = 18.75
	)
	cost := float64(u.InputTokens) / 1_000_000 * inputPer1M
	cost += float64(u.OutputTokens) / 1_000_000 * outputPer1M
	cost += float64(u.CacheRead) / 1_000_000 * cacheReadPer1M
	cost += float64(u.CacheWrite) / 1_000_000 * cacheWritePer1M
	return cost
}

// encodePath converts a filesystem path to the Claude projects directory encoding.
// Claude Code replaces both "/" and "." with "-":
// "/Users/foo/.worktrees/x" → "-Users-foo--worktrees-x"
var pathEncoder = strings.NewReplacer(string(os.PathSeparator), "-", ".", "-")

func encodePath(path string) string {
	return pathEncoder.Replace(path)
}

// FormatTokens formats a token count into a short human-readable string.
func FormatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func containsBytes(haystack, needle []byte) bool {
	return strings.Contains(string(haystack), string(needle))
}
