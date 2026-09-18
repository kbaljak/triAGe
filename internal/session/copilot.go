package session

import (
	"os"
	"path/filepath"
)

// CopilotProvider discovers sessions for GitHub Copilot CLI.
//
// Each session gets its own directory under ~/.copilot/session-state/<id>/,
// holding a workspace.yaml metadata file and an events.jsonl log, indexed by
// a SQLite database at ~/.copilot/session-store.db. Resuming is `-r` /
// `--resume <id>`.
//
// workspace.yaml's exact schema isn't documented, so this reads it with a
// tiny flat key:value scanner (see parseFlatYAML) and tries a few plausible
// key names, falling back to directory name/mtime if none match. Deleting a
// session removes its directory but can't update session-store.db (no
// SQLite driver in this project) — Copilot CLI re-scans session-state/ on
// its own, so a stale index entry should self-heal rather than error.
type CopilotProvider struct {
	home string // ~/.copilot
}

func NewCopilotProvider(override string) *CopilotProvider {
	return &CopilotProvider{home: resolveHome(override, ".copilot")}
}

func (p *CopilotProvider) ID() string   { return "copilot" }
func (p *CopilotProvider) Name() string { return "GitHub Copilot CLI" }

func (p *CopilotProvider) Detect() bool {
	return dirExists(p.home)
}

func (p *CopilotProvider) ListSessions() ([]Session, error) {
	stateDir := filepath.Join(p.home, "session-state")
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(stateDir, e.Name())

		title, cwd, updated := "", "", latestModTime([]string{dir})
		if data, err := os.ReadFile(filepath.Join(dir, "workspace.yaml")); err == nil {
			kv := parseFlatYAML(data)
			for _, k := range []string{"title", "name"} {
				if v := kv[k]; v != "" {
					title = v
					break
				}
			}
			for _, k := range []string{"cwd", "workspace", "directory", "path"} {
				if v := kv[k]; v != "" {
					cwd = v
					break
				}
			}
			for _, k := range []string{"updated_at", "updatedAt", "timestamp", "created_at", "createdAt"} {
				if v := kv[k]; v != "" {
					if t := parseAnyTime(v); !t.IsZero() {
						updated = t
					}
					break
				}
			}
		}
		if title == "" {
			title = "Untitled session"
		}
		if cwd == "" {
			cwd = "(unknown project)"
		}

		sessions = append(sessions, Session{
			ID:        e.Name(),
			Title:     title,
			Project:   cwd,
			UpdatedAt: updated,
			SizeBytes: pathSize(dir),
			Paths:     []string{dir},
		})
	}
	return sessions, nil
}

func (p *CopilotProvider) DeleteSession(s Session) error {
	return removeAll(s.Paths)
}

// ResumeCommand runs `copilot --resume <id>`.
func (p *CopilotProvider) ResumeCommand(s Session) ([]string, string, error) {
	dir := s.Project
	if !dirExists(dir) {
		dir = ""
	}
	return []string{"copilot", "--resume", s.ID}, dir, nil
}
