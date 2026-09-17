package session

import "path/filepath"

// CopilotProvider detects GitHub Copilot CLI's home directory. As of this
// writing the CLI itself only keeps process logs and IDE lock files there
// (~/.copilot/logs, ~/.copilot/ide) — no listable per-conversation history,
// since chat sessions live inside whichever editor's own workspace storage
// started them. This provider is included so Copilot still shows up in the
// agent list (truthfully, with zero sessions) instead of silently vanishing,
// and so a real session source can be wired in here later without touching
// the rest of the app.
type CopilotProvider struct {
	home string // ~/.copilot
}

func NewCopilotProvider() *CopilotProvider {
	return &CopilotProvider{home: filepath.Join(homeDir(), ".copilot")}
}

func (p *CopilotProvider) ID() string   { return "copilot" }
func (p *CopilotProvider) Name() string { return "GitHub Copilot CLI" }

func (p *CopilotProvider) Detect() bool {
	return dirExists(p.home)
}

func (p *CopilotProvider) ListSessions() ([]Session, error) {
	return nil, nil
}

func (p *CopilotProvider) DeleteSession(s Session) error {
	return removeAll(s.Paths)
}
