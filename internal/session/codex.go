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
// Sessions live at ~/.codex/sessions/YYYY/MM/DD/rollout-<timestamp>-<uuid>.jsonl
// — confirmed directly against a real Codex CLI 0.154.0 install. The first
// line of each file is a "session_meta" event whose payload carries the
// real session_id (needed for `codex resume <id>` — it's the trailing UUID,
// not the whole filename) and cwd; user turns are "user_message" events.
//
// Codex also gives sessions a human-readable name (auto-assigned, and
// presumably user-renameable — `codex resume`/`archive`/`delete` all accept
// "id or session name" per `codex resume --help`), tracked separately in an
// append-only index at ~/.codex/session_index.jsonl as {id, thread_name,
// updated_at} — also confirmed directly. That's a much better title source
// than guessing from the first user message, so it takes priority.
//
// Falls back to filename/mtime/first-user-message wherever a file doesn't
// match this shape, e.g. on an older Codex version without session_index.jsonl.
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
	names := p.readThreadNames() // session id -> {name, updated_at}, best-effort

	var sessions []Session
	root := filepath.Join(p.home, "sessions")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries rather than aborting the whole scan
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		fallbackID := strings.TrimSuffix(info.Name(), ".jsonl")
		meta := extractCodexMeta(path)
		id := meta.sessionID
		if id == "" {
			id = fallbackID
		}
		title := meta.title
		updated := info.ModTime()
		if n, ok := names[id]; ok && n.name != "" {
			title = n.name // Codex's own session name beats a guess from the first message
			if t := parseAnyTime(n.updatedAt); !t.IsZero() {
				updated = t
			}
		}
		if title == "" {
			title = fallbackID
		}
		cwd := meta.cwd
		if cwd == "" {
			cwd = "(unknown project)"
		}
		sessions = append(sessions, Session{
			ID:        id,
			Title:     title,
			Project:   cwd,
			UpdatedAt: updated,
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

type codexThreadInfo struct {
	name      string
	updatedAt string
}

// readThreadNames parses ~/.codex/session_index.jsonl once per listing: an
// append-only log of session-id -> friendly-name updates (Codex rewrites
// the name over a session's life, e.g. auto-naming it and then letting it
// evolve), so later lines for the same id win. Missing file (older Codex
// versions) just yields an empty map — callers already fall back cleanly.
func (p *CodexProvider) readThreadNames() map[string]codexThreadInfo {
	out := map[string]codexThreadInfo{}
	f, err := os.Open(filepath.Join(p.home, "session_index.jsonl"))
	if err != nil {
		return out
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 64*1024)
	for {
		line, readErr := r.ReadString('\n')
		if len(line) > 0 {
			var e struct {
				ID         string `json:"id"`
				ThreadName string `json:"thread_name"`
				UpdatedAt  string `json:"updated_at"`
			}
			if json.Unmarshal([]byte(line), &e) == nil && e.ID != "" && e.ThreadName != "" {
				out[e.ID] = codexThreadInfo{name: e.ThreadName, updatedAt: e.UpdatedAt}
			}
		}
		if readErr != nil {
			break
		}
	}
	return out
}

func (p *CodexProvider) DeleteSession(s Session) error {
	return removeAll(s.Paths)
}

// ResumeCommand runs `codex resume <session-id-or-name>`, documented at
// https://learn.chatgpt.com/docs/codex/cli and confirmed independently via
// GitHub discussions/issues on openai/codex. Unverified end-to-end since
// Codex isn't installed on the machine this was built on.
func (p *CodexProvider) ResumeCommand(s Session) ([]string, string, error) {
	dir := s.Project
	if !dirExists(dir) {
		dir = ""
	}
	return []string{"codex", "resume", s.ID}, dir, nil
}

type codexMeta struct {
	sessionID string
	cwd       string
	title     string
}

// codexEnvelope is the outer shape of every rollout line: {"type":...,
// "item":{...}} in current Codex CLI versions, but some released versions
// used "payload" for the same wrapper — accept either rather than betting
// on one, consistent with how the rest of this provider degrades gracefully.
type codexEnvelope struct {
	Type    string          `json:"type"`
	Item    json.RawMessage `json:"item"`
	Payload json.RawMessage `json:"payload"`
}

func (e codexEnvelope) data() json.RawMessage {
	if len(e.Item) > 0 {
		return e.Item
	}
	return e.Payload
}

// extractCodexMeta streams a rollout file (never loading it fully — these
// can grow large over a long session) looking for the session_meta event
// (session_id, cwd) and the first user_message event (for a title fallback,
// since Codex rollouts carry no title/summary field of their own).
func extractCodexMeta(path string) codexMeta {
	f, err := os.Open(path)
	if err != nil {
		return codexMeta{}
	}
	defer f.Close()

	var meta codexMeta
	r := bufio.NewReaderSize(f, 64*1024)
	for {
		line, readErr := r.ReadString('\n')
		if len(line) > 0 {
			var env codexEnvelope
			if json.Unmarshal([]byte(line), &env) == nil {
				switch env.Type {
				case "session_meta":
					var m struct {
						SessionID string `json:"session_id"`
						ID        string `json:"id"`
						Cwd       string `json:"cwd"`
					}
					if json.Unmarshal(env.data(), &m) == nil {
						if m.SessionID != "" {
							meta.sessionID = m.SessionID
						} else {
							meta.sessionID = m.ID
						}
						meta.cwd = m.Cwd
					}
				case "user_message":
					if meta.title == "" {
						var m struct {
							Content string `json:"content"`
						}
						if json.Unmarshal(env.data(), &m) == nil {
							meta.title = cleanFallbackTitle(m.Content)
						}
					}
				}
			}
		}
		if readErr != nil {
			break
		}
		if meta.sessionID != "" && meta.cwd != "" && meta.title != "" {
			break
		}
	}
	return meta
}
