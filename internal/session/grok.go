package session

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GrokProvider discovers sessions for xAI's "Grok Build" (the `grok` CLI
// terminal coding agent — distinct from the Grok chatbot, and from
// unofficial community projects also named "grok-cli").
//
// Sessions live at ~/.grok/sessions/<encoded-cwd>/<session-id>/, each with a
// summary.json metadata file alongside the actual transcript files
// (updates.jsonl, chat_history.jsonl, etc.) that make up the rest of the
// session directory. Resuming is `grok --resume <id-or-title>` — UUID-shaped
// values are always treated as an ID, anything else matches against titles.
//
// GROK_HOME can relocate the whole ~/.grok tree; not handled here.
type GrokProvider struct {
	home string // ~/.grok
}

func NewGrokProvider() *GrokProvider {
	return &GrokProvider{home: filepath.Join(homeDir(), ".grok")}
}

func (p *GrokProvider) ID() string   { return "grok" }
func (p *GrokProvider) Name() string { return "Grok Build (xAI)" }

func (p *GrokProvider) Detect() bool {
	return dirExists(filepath.Join(p.home, "sessions"))
}

func (p *GrokProvider) ListSessions() ([]Session, error) {
	root := filepath.Join(p.home, "sessions")
	cwdDirs, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, cwdDir := range cwdDirs {
		if !cwdDir.IsDir() {
			continue
		}
		cwdPath := filepath.Join(root, cwdDir.Name())
		sessionDirs, err := os.ReadDir(cwdPath)
		if err != nil {
			continue
		}
		for _, sd := range sessionDirs {
			if !sd.IsDir() {
				continue
			}
			sessionDir := filepath.Join(cwdPath, sd.Name())

			// cwd/id are nested under an "info" object here, not top-level:
			//   {"info":{"id":"...","cwd":"..."},"created_at":"...",
			//    "updated_at":"...","generated_title":"...",
			//    "session_summary":"...", ...}
			var meta struct {
				Info struct {
					Cwd string `json:"cwd"`
				} `json:"info"`
				CreatedAt      string `json:"created_at"`
				UpdatedAt      string `json:"updated_at"`
				GeneratedTitle string `json:"generated_title"`
				SessionSummary string `json:"session_summary"`
			}
			if data, err := os.ReadFile(filepath.Join(sessionDir, "summary.json")); err == nil {
				_ = json.Unmarshal(data, &meta)
			}

			title := meta.GeneratedTitle
			if title == "" {
				title = meta.SessionSummary
			}
			if title == "" {
				title = "Untitled session"
			}
			cwd := meta.Info.Cwd
			if cwd == "" {
				cwd = "(unknown project)"
			}
			updated := parseAnyTime(meta.UpdatedAt, meta.CreatedAt)
			if updated.IsZero() {
				updated = latestModTime([]string{sessionDir})
			}

			sessions = append(sessions, Session{
				// The directory name is the session id by construction —
				// more certain than trusting the JSON content to match it.
				ID:        sd.Name(),
				Title:     title,
				Project:   cwd,
				UpdatedAt: updated,
				SizeBytes: pathSize(sessionDir),
				Paths:     []string{sessionDir},
			})
		}
	}
	return sessions, nil
}

func (p *GrokProvider) DeleteSession(s Session) error {
	return removeAll(s.Paths)
}

// ResumeCommand runs `grok --resume <id>`.
func (p *GrokProvider) ResumeCommand(s Session) ([]string, string, error) {
	dir := s.Project
	if !dirExists(dir) {
		dir = ""
	}
	return []string{"grok", "--resume", s.ID}, dir, nil
}
