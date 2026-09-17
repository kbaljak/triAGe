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
// The Gemini CLI keeps per-project state under ~/.gemini/tmp/<project-hash>/,
// including saved chat checkpoints. Field names here are best-effort: this
// machine only has Antigravity installed, not the plain Gemini CLI, so this
// provider degrades gracefully (falls back to filename/mtime, never errors
// out) if a checkpoint's internal shape doesn't match what's expected.
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

			title := strings.TrimSuffix(strings.TrimPrefix(f.Name(), "checkpoint-"), ".json")
			if data, err := os.ReadFile(path); err == nil {
				var probe struct {
					Title string `json:"title"`
					Tag   string `json:"tag"`
				}
				if json.Unmarshal(data, &probe) == nil {
					if probe.Title != "" {
						title = probe.Title
					} else if probe.Tag != "" {
						title = probe.Tag
					}
				}
			}

			sessions = append(sessions, Session{
				ID:        proj.Name() + "/" + f.Name(),
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
