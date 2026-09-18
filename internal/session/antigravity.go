package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AntigravityProvider discovers sessions for Google's Antigravity CLI, which
// nests under the Gemini CLI home directory (~/.gemini/antigravity-cli)
// rather than having a directory of its own.
//
// Conversation metadata (title, preview, workspace, timestamps) lives in a
// single JSON index at cache/conversation_metadata.json; the conversation
// content itself is a sqlite DB per conversation ID under conversations/,
// with matching per-ID files under annotations/, presence/ and brain/.
type AntigravityProvider struct {
	home string // ~/.gemini/antigravity-cli
}

func NewAntigravityProvider() *AntigravityProvider {
	return &AntigravityProvider{home: filepath.Join(homeDir(), ".gemini", "antigravity-cli")}
}

func (p *AntigravityProvider) ID() string   { return "antigravity" }
func (p *AntigravityProvider) Name() string { return "Antigravity" }

func (p *AntigravityProvider) Detect() bool {
	return dirExists(p.home)
}

func (p *AntigravityProvider) metadataPath() string {
	return filepath.Join(p.home, "cache", "conversation_metadata.json")
}

func (p *AntigravityProvider) lastConversationsPath() string {
	return filepath.Join(p.home, "cache", "last_conversations.json")
}

type antigravityConversationSummary struct {
	ID            string   `json:"ID"`
	Title         string   `json:"Title"`
	Preview       string   `json:"Preview"`
	NumSteps      int      `json:"NumSteps"`
	UpdatedAt     string   `json:"UpdatedAt"`
	WorkspaceURIs []string `json:"WorkspaceURIs"`
	AgentName     string   `json:"AgentName"`
}

type antigravityConversationEntry struct {
	Summary          antigravityConversationSummary `json:"summary"`
	IsInternal       bool                           `json:"is_internal"`
	LastModifiedTime string                         `json:"last_modified_time"`
}

type antigravityMetadataFile struct {
	Conversations map[string]antigravityConversationEntry `json:"conversations"`
}

func (p *AntigravityProvider) readMetadata() (antigravityMetadataFile, error) {
	var meta antigravityMetadataFile
	data, err := os.ReadFile(p.metadataPath())
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	if meta.Conversations == nil {
		meta.Conversations = map[string]antigravityConversationEntry{}
	}
	return meta, nil
}

func (p *AntigravityProvider) ListSessions() ([]Session, error) {
	meta, err := p.readMetadata()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for id, entry := range meta.Conversations {
		title := strings.TrimSpace(entry.Summary.Title)
		if title == "" {
			title = strings.TrimSpace(entry.Summary.Preview)
			if len(title) > 70 {
				title = title[:70] + "…"
			}
		}
		if title == "" {
			title = "Untitled conversation"
		}

		project := "(unknown workspace)"
		if len(entry.Summary.WorkspaceURIs) > 0 {
			project = strings.TrimPrefix(entry.Summary.WorkspaceURIs[0], "file://")
		}

		updated := parseAnyTime(entry.Summary.UpdatedAt, entry.LastModifiedTime)

		paths := p.sessionPaths(id)
		var size int64
		for _, path := range paths {
			size += pathSize(path)
		}
		if updated.IsZero() {
			updated = latestModTime(paths)
		}

		sessions = append(sessions, Session{
			ID:        id,
			Title:     title,
			Project:   project,
			UpdatedAt: updated,
			SizeBytes: size,
			Paths:     paths,
		})
	}
	return sessions, nil
}

// sessionPaths returns every on-disk path that belongs to a conversation ID,
// limited to the ones that actually exist.
func (p *AntigravityProvider) sessionPaths(id string) []string {
	candidates := []string{
		filepath.Join(p.home, "conversations", id+".db"),
		filepath.Join(p.home, "annotations", id+".pbtxt"),
		filepath.Join(p.home, "presence", id+".lock"),
		filepath.Join(p.home, "brain", id),
	}
	var paths []string
	for _, c := range candidates {
		if _, err := os.Lstat(c); err == nil {
			paths = append(paths, c)
		}
	}
	return paths
}

func (p *AntigravityProvider) DeleteSession(s Session) error {
	// Remove the conversation's own files first.
	fileErr := removeAll(s.Paths)

	// Then drop it from the two JSON indexes so it stops showing up in
	// Antigravity's own conversation list too.
	meta, err := p.readMetadata()
	if err == nil {
		if _, ok := meta.Conversations[s.ID]; ok {
			delete(meta.Conversations, s.ID)
			_ = writeJSONFile(p.metadataPath(), meta)
		}
	}

	if data, err := os.ReadFile(p.lastConversationsPath()); err == nil {
		var last map[string]string
		if json.Unmarshal(data, &last) == nil {
			changed := false
			for workspace, convID := range last {
				if convID == s.ID {
					delete(last, workspace)
					changed = true
				}
			}
			if changed {
				_ = writeJSONFile(p.lastConversationsPath(), last)
			}
		}
	}

	return fileErr
}

// parseAnyTime tries each candidate timestamp string in order (Antigravity
// uses RFC3339-with-nanos for one field and an offset form for the other)
// and returns the first that parses.
func parseAnyTime(candidates ...string) time.Time {
	layouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, c); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func latestModTime(paths []string) time.Time {
	var latest time.Time
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && fi.ModTime().After(latest) {
			latest = fi.ModTime()
		}
	}
	return latest
}

func writeJSONFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
