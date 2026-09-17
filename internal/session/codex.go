package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// CodexProvider discovers sessions for OpenAI's Codex CLI, the local coding
// agent that persists conversation "rollouts" to disk (this is the actual
// on-disk artifact behind "ChatGPT" as a terminal coding agent; the ChatGPT
// web/desktop apps don't expose local session files at all).
//
// Sessions live at ~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl. Codex isn't
// installed on the machine this was built on, so field names are best-effort
// from the known rollout format; extraction degrades to filename/mtime
// rather than failing if a line doesn't parse as expected.
type CodexProvider struct {
	home string // ~/.codex
}

func NewCodexProvider() *CodexProvider {
	return &CodexProvider{home: filepath.Join(homeDir(), ".codex")}
}

func (p *CodexProvider) ID() string   { return "codex" }
func (p *CodexProvider) Name() string { return "Codex (OpenAI)" }

func (p *CodexProvider) Detect() bool {
	return dirExists(filepath.Join(p.home, "sessions"))
}

func (p *CodexProvider) ListSessions() ([]Session, error) {
	var sessions []Session
	root := filepath.Join(p.home, "sessions")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries rather than aborting the whole scan
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		title, cwd := extractCodexMeta(path)
		if title == "" {
			title = strings.TrimSuffix(info.Name(), ".jsonl")
		}
		if cwd == "" {
			cwd = "(unknown project)"
		}
		sessions = append(sessions, Session{
			ID:        strings.TrimSuffix(info.Name(), ".jsonl"),
			Title:     title,
			Project:   cwd,
			UpdatedAt: info.ModTime(),
			SizeBytes: info.Size(),
			Paths:     []string{path},
		})
		return nil
	})
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	}
	return sessions, nil
}

func (p *CodexProvider) DeleteSession(s Session) error {
	return removeAll(s.Paths)
}

// ResumeCommand is best-effort like the rest of this provider: `codex resume
// <id>` is the documented subcommand, but it's unverified since Codex isn't
// installed on the machine this was built on.
func (p *CodexProvider) ResumeCommand(s Session) ([]string, string, error) {
	dir := s.Project
	if !dirExists(dir) {
		dir = ""
	}
	return []string{"codex", "resume", s.ID}, dir, nil
}

func extractCodexMeta(path string) (title, cwd string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 64*1024)
	for {
		line, readErr := r.ReadString('\n')
		if len(line) > 0 {
			switch {
			case cwd == "" && strings.Contains(line, `"cwd":"`):
				var m struct {
					Payload struct {
						Cwd string `json:"cwd"`
					} `json:"payload"`
					Cwd string `json:"cwd"`
				}
				if json.Unmarshal([]byte(line), &m) == nil {
					if m.Payload.Cwd != "" {
						cwd = m.Payload.Cwd
					} else {
						cwd = m.Cwd
					}
				}
			case title == "" && strings.Contains(line, `"role":"user"`):
				var m struct {
					Content string `json:"content"`
					Text    string `json:"text"`
				}
				if json.Unmarshal([]byte(line), &m) == nil {
					c := m.Content
					if c == "" {
						c = m.Text
					}
					c = strings.TrimSpace(c)
					if len(c) > 70 {
						c = c[:70] + "…"
					}
					title = c
				}
			}
		}
		if readErr != nil {
			break
		}
		if title != "" && cwd != "" {
			break
		}
	}
	return title, cwd
}
