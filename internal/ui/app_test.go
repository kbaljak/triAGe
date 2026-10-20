package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kbaljak/triAGe/internal/config"
	"github.com/kbaljak/triAGe/internal/session"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// newTestModel builds a minimal Model with one agent in its list, without
// going through New() (which scans every real provider on the machine).
// Redirects config.Save()'s target to a temp directory, since this model's
// flows (e.g. adding a profile) really do call Save() — without this,
// a test would overwrite the real ~/.config/triage/config.json.
func newTestModel(t *testing.T, agentID string) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p := session.NewProvider(agentID, t.TempDir())
	agentList := list.New([]list.Item{agentItem{provider: p, detected: p.Detect()}}, agentDelegate{}, 80, 20)
	profileList := list.New(nil, profileDelegate{}, 80, 20)
	sessionList := list.New(nil, sessionDelegate{}, 80, 20)
	return Model{
		state:       viewAgents,
		cfg:         config.Config{AgentPaths: map[string]string{}},
		agentList:   agentList,
		profileList: profileList,
		sessionList: sessionList,
		marked:      map[string]bool{},
		width:       80,
		height:      24,
	}
}

// TestSelectAgentAlwaysShowsProfileList drives the flow exactly as
// described: pick an agent from the agent list, land on its profile list
// (showing just "Default" the first time), then add a new profile from
// there. Locks in that "enter" on the agent list always opens the profile
// list — it must never jump straight to sessions, even when there's only
// the one Default profile so far, or there'd be no way to ever discover or
// use "n" to create a second one.
func TestSelectAgentAlwaysShowsProfileList(t *testing.T) {
	m := newTestModel(t, "codex")

	// Selecting the (only) agent in the list must land on the profile
	// list, not skip straight to sessions.
	mi, cmd := m.Update(key("enter"))
	m = mi.(Model)
	if cmd != nil {
		t.Fatalf("unexpected cmd from selecting the agent: %v", cmd)
	}
	if m.state != viewProfiles {
		t.Fatalf("state = %v, want viewProfiles", m.state)
	}
	if m.profileAgentID != "codex" {
		t.Fatalf("profileAgentID = %q, want %q", m.profileAgentID, "codex")
	}
	items := m.profileList.Items()
	if len(items) != 1 || items[0].(profileItem).name != "Default" {
		t.Fatalf("profileList = %+v, want just [Default]", items)
	}

	// "n" on the profile list starts the add-profile flow.
	mi, _ = m.Update(key("n"))
	m = mi.(Model)
	if m.state != viewNewProfileName {
		t.Fatalf("state = %v, want viewNewProfileName", m.state)
	}

	// Type a name and submit.
	for _, r := range "personal" {
		mi, _ = m.Update(key(string(r)))
		m = mi.(Model)
	}
	mi, _ = m.Update(key("enter"))
	m = mi.(Model)
	if m.state != viewSetPath {
		t.Fatalf("state = %v, want viewSetPath after naming the profile", m.state)
	}
	if !m.pathIsNewProfile || m.pathEditProfile != "personal" {
		t.Fatalf("pathIsNewProfile=%v pathEditProfile=%q, want true/\"personal\"", m.pathIsNewProfile, m.pathEditProfile)
	}

	// Type a (not-yet-existing) path and submit — should get created.
	path := t.TempDir() + "/fresh-profile-dir"
	for _, r := range path {
		mi, _ = m.Update(key(string(r)))
		m = mi.(Model)
	}
	mi, _ = m.Update(key("enter"))
	m = mi.(Model)

	if m.pathError != "" {
		t.Fatalf("unexpected path error: %s", m.pathError)
	}
	if m.state != viewProfiles {
		t.Fatalf("state = %v, want viewProfiles after finishing profile creation", m.state)
	}

	// The whole point: the profile is now actually there, both in config
	// and in the rendered list (Default + the new one).
	if len(m.cfg.Profiles["codex"]) != 1 || m.cfg.Profiles["codex"][0].Name != "personal" {
		t.Fatalf("cfg.Profiles[codex] = %+v, want one profile named personal", m.cfg.Profiles["codex"])
	}
	items = m.profileList.Items()
	if len(items) != 2 {
		t.Fatalf("profileList has %d items, want 2 (Default + personal)", len(items))
	}
	names := []string{items[0].(profileItem).name, items[1].(profileItem).name}
	if names[0] != "Default" || names[1] != "personal" {
		t.Fatalf("profile names = %v, want [Default personal]", names)
	}
}

// TestNewSessionSupported drives 'n' on the session list for a provider
// that implements NewSessionStarter (claude does, confirmed against its
// real --help output): it should open the prompt and, on submit, return a
// non-nil tea.Cmd without erroring. The returned cmd (an ExecProcess
// wrapper) is never invoked here — asserting it's non-nil and that no
// validation error was set is enough to confirm the wiring is correct
// without actually spawning the real CLI.
func TestNewSessionSupported(t *testing.T) {
	m := newTestModel(t, "claude")
	p := m.agentList.Items()[0].(agentItem).provider
	m.openAgent(p, viewAgents)

	m.beginNewSession()
	if m.state != viewNewSession {
		t.Fatalf("state = %v, want viewNewSession", m.state)
	}

	cmd := m.submitNewSession()
	if m.newSessionError != "" {
		t.Fatalf("unexpected error: %s", m.newSessionError)
	}
	if cmd == nil {
		t.Fatal("submitNewSession returned a nil cmd, want a tea.ExecProcess command")
	}
	if m.state != viewSessions {
		t.Fatalf("state = %v, want viewSessions after starting a new session", m.state)
	}
}

// TestNewSessionUnsupported checks a provider with no NewSessionStarter
// (Antigravity has none) refuses cleanly with a status message, rather than
// opening a prompt that could never succeed.
func TestNewSessionUnsupported(t *testing.T) {
	m := newTestModel(t, "antigravity")
	p := m.agentList.Items()[0].(agentItem).provider
	m.openAgent(p, viewAgents)

	m.beginNewSession()
	if m.state == viewNewSession {
		t.Fatal("beginNewSession opened the prompt for a provider that doesn't support it")
	}
	if m.status == "" {
		t.Error("expected a status message explaining new sessions aren't supported")
	}
}
