package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// GeminiProvider discovers sessions for the plain Gemini CLI (as opposed to
// Antigravity, which is a separate agent that happens to share the same
// ~/.gemini home directory on machines that have both installed).
//
// Confirmed via https://geminicli.com/docs/cli/session-management/ and
// https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/session-management.md:
// auto-saved, resumable sessions live at
// ~/.gemini/tmp/<project-hash>/chats/, and are resumed with
// `gemini --resume <session-id>`. Gemini CLI's docs describe named
// "chat checkpoints" (`/chat save <tag>`) as a related but separate
// feature, resumed with an in-session slash command rather than a launch
// flag — those aren't handled here since they're not resumable the same
// way every other session in this app is.
//
// Neither the exact JSON filename convention nor the internal schema for
// entries under chats/ is documented, and the plain Gemini CLI isn't
// installed on the machine this was built on (only Antigravity, which uses
// its own antigravity-cli subdirectory of ~/.gemini). So this degrades
// gracefully — falls back to filename/mtime — if a file's shape doesn't
// match what's guessed here.
type GeminiProvider struct {
	home string // ~/.gemini
}

func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{home: homeDir() + "/.gemini"}
}

func (p *GeminiProvider) ID() string   { return "gemini" }
func (p *GeminiProvider) Name() string { return "Gemini CLI" }

func (p *GeminiProvider) Detect() bool {
	return dirExists(filepath.Join(p.home, "tmp"))
}

func (p *GeminiProvider) ListSessions() ([]Session, error) {
	tmpDir := filepath.Join(p.home, "tmp")
	projectDirs, err := os.ReadDir(tmpDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, proj := range projectDirs {
		if !proj.IsDir() {
			continue
		}
		projectLabel := "(project hash: " + proj.Name() + ")"
		chatsDir := filepath.Join(tmpDir, proj.Name(), "chats")
		files, err := os.ReadDir(chatsDir)
		if err != nil {
			continue // no saved chats for this project
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			path := filepath.Join(chatsDir, f.Name())
			info, err := f.Info()
			if err != nil {
				continue
			}

			// The session id `--resume` expects is the filename stem
			// (docs show `gemini --resume <uuid>`, and every other agent
			// this app supports names its session files by session id the
			// same way) — track it separately from the display title.
			id := strings.TrimSuffix(f.Name(), ".json")
			title := strings.TrimSuffix(strings.TrimPrefix(f.Name(), "checkpoint-"), ".json")
			if data, err := os.ReadFile(path); err == nil {
				var probe struct {
					Title     string `json:"title"`
					Tag       string `json:"tag"`
					Name      string `json:"name"`
					SessionID string `json:"sessionId"`
				}
				if json.Unmarshal(data, &probe) == nil {
					switch {
					case probe.Title != "":
						title = probe.Title
					case probe.Tag != "":
						title = probe.Tag
					case probe.Name != "":
						title = probe.Name
					}
					if probe.SessionID != "" {
						id = probe.SessionID
					}
				}
			}

			sessions = append(sessions, Session{
				ID:        id,
				Title:     title,
				Project:   projectLabel,
				UpdatedAt: info.ModTime(),
				SizeBytes: info.Size(),
				Paths:     []string{path},
			})
		}
	}
	return sessions, nil
}

func (p *GeminiProvider) DeleteSession(s Session) error {
	return removeAll(s.Paths)
}

// ResumeCommand runs `gemini --resume <session-id>`, confirmed directly in
// Gemini CLI's own docs (see the package doc comment for links). Best-effort
// like the rest of this provider: relies on the session id resolved in
// ListSessions actually being what --resume expects.
func (p *GeminiProvider) ResumeCommand(s Session) ([]string, string, error) {
	return []string{"gemini", "--resume", s.ID}, "", nil
}
