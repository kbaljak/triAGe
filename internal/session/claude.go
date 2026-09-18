package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ClaudeProvider discovers sessions for Claude Code (the `claude` CLI).
//
// Claude Code stores one JSON-lines transcript per session at
// ~/.claude/projects/<encoded-cwd>/<session-uuid>.jsonl. Each line is a
// distinct event; we only care about a handful of event types to recover a
// human title and the working directory the session ran in.
type ClaudeProvider struct {
	home string // ~/.claude
}

func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{home: filepath.Join(homeDir(), ".claude")}
}

func (p *ClaudeProvider) ID() string   { return "claude" }
func (p *ClaudeProvider) Name() string { return "Claude Code" }

func (p *ClaudeProvider) Detect() bool {
	return dirExists(p.home)
}

func (p *ClaudeProvider) ListSessions() ([]Session, error) {
	projectsDir := filepath.Join(p.home, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, projEntry := range entries {
		if !projEntry.IsDir() {
			continue
		}
		projDir := filepath.Join(projectsDir, projEntry.Name())
		files, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			path := filepath.Join(projDir, f.Name())
			info, err := f.Info()
			if err != nil {
				continue
			}
			id := strings.TrimSuffix(f.Name(), ".jsonl")
			title, cwd := extractClaudeMeta(path)
			if title == "" {
				title = "Untitled session"
			}
			if cwd == "" {
				cwd = "(unknown project: " + projEntry.Name() + ")"
			}
			sessions = append(sessions, Session{
				ID:        id,
				Title:     title,
				Project:   cwd,
				UpdatedAt: info.ModTime(),
				SizeBytes: info.Size(),
				Paths:     []string{path},
			})
		}
	}
	return sessions, nil
}

func (p *ClaudeProvider) DeleteSession(s Session) error {
	return removeAll(s.Paths)
}

// ResumeCommand runs `claude --resume <session-id>`, launched from the
// session's original working directory so Claude Code finds the same
// project context it started in.
func (p *ClaudeProvider) ResumeCommand(s Session) ([]string, string, error) {
	dir := s.Project
	if !dirExists(dir) {
		dir = "" // fall back to triAGe's own cwd rather than failing outright
	}
	return []string{"claude", "--resume", s.ID}, dir, nil
}

// extractClaudeMeta streams a session transcript line by line (never
// loading the whole file, since transcripts can hold multi-megabyte tool
// output lines) looking for the most recent AI-generated title and the
// working directory recorded on a user turn.
func extractClaudeMeta(path string) (title, cwd string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	var aiTitle, customTitle, fallbackTitle string
	r := bufio.NewReaderSize(f, 64*1024)
	for {
		line, readErr := r.ReadString('\n')
		if len(line) > 0 {
			switch {
			case strings.Contains(line, `"type":"custom-title"`):
				// A name the user set explicitly (e.g. via /rename) always
				// wins over anything AI-generated or guessed from a prompt.
				var m struct {
					CustomTitle string `json:"customTitle"`
				}
				if json.Unmarshal([]byte(line), &m) == nil && m.CustomTitle != "" {
					customTitle = m.CustomTitle
				}
			case strings.Contains(line, `"type":"ai-title"`):
				var m struct {
					AiTitle string `json:"aiTitle"`
				}
				if json.Unmarshal([]byte(line), &m) == nil && m.AiTitle != "" {
					aiTitle = m.AiTitle // later lines override earlier (freshest title wins)
				}
			case strings.Contains(line, `"type":"last-prompt"`):
				// Same deal as ai-title: this event is re-emitted on every
				// turn, so keep overwriting to end up with the most recent
				// prompt rather than getting stuck on the very first one.
				var m struct {
					LastPrompt string `json:"lastPrompt"`
				}
				if json.Unmarshal([]byte(line), &m) == nil && m.LastPrompt != "" {
					fallbackTitle = m.LastPrompt
				}
			case cwd == "" && strings.Contains(line, `"cwd":"`):
				// Not every event type carries a message (e.g. a cancelled
				// /resume only logs "system" events), so accept cwd from
				// any line rather than requiring a "user" turn.
				var m struct {
					Cwd string `json:"cwd"`
				}
				if json.Unmarshal([]byte(line), &m) == nil {
					cwd = m.Cwd
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	switch {
	case customTitle != "":
		title = customTitle
	case aiTitle != "":
		title = aiTitle
	default:
		title = cleanFallbackTitle(fallbackTitle)
	}
	return title, cwd
}

// cleanFallbackTitle turns a raw prompt (used as a title when Claude hasn't
// generated a real one for the session yet) into something that reads
// reasonably on a single list line: collapsed whitespace, capped length.
func cleanFallbackTitle(s string) string {
	s = strings.Join(strings.Fields(s), " ") // collapse newlines/tabs/runs of spaces
	const maxLen = 80
	if r := []rune(s); len(r) > maxLen {
		s = strings.TrimSpace(string(r[:maxLen])) + "…"
	}
	return s
}
