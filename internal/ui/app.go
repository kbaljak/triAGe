// Package ui implements triAGe's terminal GUI: a two-screen browser over
// every AI agent provider in internal/session, letting the user drill into
// an agent's sessions to resume, mark, and delete them.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kbaljak/triAGe/internal/config"
	"github.com/kbaljak/triAGe/internal/session"
)

// hostname is resolved once at startup for the agent-list title bar.
var hostname = func() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "this machine"
}()

type viewState int

const (
	viewAgents viewState = iota
	viewSessions
	viewConfirm
	viewSetPath
	viewFilter
)

// confirmTarget describes what a pending delete confirmation would delete.
type confirmTarget struct {
	sessions []session.Session
}

// Model is the root Bubble Tea model for triAGe.
type Model struct {
	state    viewState
	returnTo viewState // state to go back to after a confirm/set-path dialog

	cfg config.Config // loaded once at startup, rewritten as paths are set

	agentList   list.Model
	sessionList list.Model

	active      session.Provider  // provider currently open in the session view
	allSessions []session.Session // every session for the active provider, unfiltered
	marked      map[string]bool   // session ID -> marked, scoped to the active provider

	filterQuery string          // the active advanced-filter query, "" if none
	filterInput textinput.Model // active while state == viewFilter
	filterError string          // validation message shown under the input

	confirm *confirmTarget

	pathInput      textinput.Model // active while state == viewSetPath
	pathTargetID   string          // provider ID the path prompt is editing
	pathTargetName string          // its display name, for the prompt title
	pathError      string          // validation message shown under the input

	status string // transient status line, replaced on the next action

	width, height int
}

// New builds the initial model, eagerly scanning every provider (using any
// custom paths already saved in config) so the agent list can show session
// counts — and which agents weren't found — right away.
func New() Model {
	cfg, _ := config.Load() // a missing/unreadable config just means no overrides yet

	agentList := list.New(buildAgentItems(session.All(cfg.AgentPaths)), agentDelegate{}, 0, 0)
	agentList.Title = "AI Agents"
	agentList.SetShowTitle(false) // we draw our own title bar
	agentList.SetShowStatusBar(false)
	agentList.SetShowHelp(false)
	agentList.SetFilteringEnabled(true)
	agentList.Styles.Title = titleStyle

	sessionList := list.New(nil, sessionDelegate{}, 0, 0)
	sessionList.SetShowTitle(false)
	sessionList.SetShowStatusBar(false)
	sessionList.SetShowHelp(false)
	sessionList.SetFilteringEnabled(true)

	m := Model{
		state:       viewAgents,
		cfg:         cfg,
		agentList:   agentList,
		sessionList: sessionList,
		marked:      map[string]bool{},
	}

	none := true
	for _, it := range agentList.Items() {
		if ai, ok := it.(agentItem); ok && ai.detected {
			none = false
			break
		}
	}
	if none {
		m.status = "No agents found automatically. Highlight one and press 'p' to set its config path."
	}
	return m
}

// buildAgentItems scans every given provider and wraps it as a list item,
// including ones that aren't detected at all — they still need to show up
// so their path can be set with 'p'.
func buildAgentItems(providers []session.Provider) []list.Item {
	items := make([]list.Item, 0, len(providers))
	for _, p := range providers {
		detected := p.Detect()
		var sessionCount int
		var total int64
		var loadErr error
		if detected {
			sessions, err := p.ListSessions()
			loadErr = err
			sessionCount = len(sessions)
			for _, s := range sessions {
				total += s.SizeBytes
			}
		}
		items = append(items, agentItem{
			provider:     p,
			detected:     detected,
			sessionCount: sessionCount,
			totalSize:    total,
			loadErr:      loadErr,
		})
	}
	return items
}

func (m Model) Init() tea.Cmd { return nil }

// resumeFinishedMsg arrives after tea.ExecProcess hands control back from
// the agent CLI it launched (see beginResume).
type resumeFinishedMsg struct {
	sessionTitle string
	err          error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		listW, listH := m.listSize()
		m.agentList.SetSize(listW, listH)
		m.sessionList.SetSize(listW, listH)
		return m, nil

	case resumeFinishedMsg:
		if msg.err != nil {
			m.status = errorStyle.Render("Resume of " + msg.sessionTitle + " exited with an error: " + msg.err.Error())
		} else {
			m.status = "Welcome back — resumed " + msg.sessionTitle + "."
		}
		if m.active != nil {
			m.loadSessions(m.active) // pick up whatever changed while it ran
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case viewAgents:
			return m.updateAgents(msg)
		case viewSessions:
			return m.updateSessions(msg)
		case viewConfirm:
			return m.updateConfirm(msg)
		case viewSetPath:
			return m.updateSetPath(msg)
		case viewFilter:
			return m.updateFilter(msg)
		}
		return m, nil
	}

	// Anything else (the list's async filter-match results, a text input's
	// cursor blink, etc.) still needs to reach whichever list — or text
	// input — is active, or it silently never finishes applying.
	var cmd tea.Cmd
	switch m.state {
	case viewAgents:
		m.agentList, cmd = m.agentList.Update(msg)
	case viewSessions:
		m.sessionList, cmd = m.sessionList.Update(msg)
	case viewSetPath:
		m.pathInput, cmd = m.pathInput.Update(msg)
	case viewFilter:
		m.filterInput, cmd = m.filterInput.Update(msg)
	}
	return m, cmd
}

// listSize reserves room for our own title bar, a blank line and the status
// bar so the embedded list never draws outside its allotted rows.
func (m Model) listSize() (int, int) {
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	w := m.width
	if w < 1 {
		w = 1
	}
	return w, h
}

func (m Model) updateAgents(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.agentList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.agentList, cmd = m.agentList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "enter":
		it, ok := m.agentList.SelectedItem().(agentItem)
		if !ok {
			return m, nil
		}
		if !it.detected {
			m.status = it.provider.Name() + " isn't detected — press 'p' to set its config path."
			return m, nil
		}
		m.openAgent(it.provider)
		return m, nil
	case "p":
		m.beginSetPath()
		return m, nil
	case "r":
		m.reloadAgents()
		m.status = "Refreshed agent list."
		return m, nil
	}

	var cmd tea.Cmd
	m.agentList, cmd = m.agentList.Update(msg)
	return m, cmd
}

func (m Model) updateSessions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sessionList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.sessionList, cmd = m.sessionList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "left", "backspace":
		m.state = viewAgents
		m.status = ""
		return m, nil
	case "enter":
		return m, m.beginResume()
	case " ":
		return m, m.toggleMark()
	case "d", "delete":
		m.beginDelete()
		return m, nil
	case "f":
		m.beginFilter()
		return m, nil
	case "r":
		if m.active != nil {
			m.loadSessions(m.active)
			m.status = "Refreshed sessions."
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.sessionList, cmd = m.sessionList.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.performDelete()
		m.state = m.returnTo
		return m, nil
	case "n", "N", "esc", "q", "ctrl+c":
		m.confirm = nil
		m.state = m.returnTo
		m.status = "Cancelled."
		return m, nil
	}
	return m, nil
}

func (m Model) updateSetPath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.state = m.returnTo
		return m, nil
	case "enter":
		m.submitSetPath()
		return m, nil
	}
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	m.pathError = ""
	return m, cmd
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.state = m.returnTo
		return m, nil
	case "enter":
		m.submitFilter()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterError = ""
	return m, cmd
}

// openAgent switches into the session view for the given provider, loading
// (or reloading) its sessions and clearing any stale marks and filter.
func (m *Model) openAgent(p session.Provider) {
	m.active = p
	m.marked = map[string]bool{}
	m.filterQuery = ""
	m.loadSessions(p)
	m.state = viewSessions
	m.status = ""
}

func (m *Model) loadSessions(p session.Provider) {
	sessions, err := p.ListSessions()
	m.allSessions = sessions
	m.applyFilter()
	if err != nil {
		m.status = errorStyle.Render("Failed to load sessions: " + err.Error())
	}
}

// applyFilter rebuilds the session list's items from m.allSessions,
// keeping only the ones matching the active filter query (all of them, if
// there is none), and preserving marks by session ID.
func (m *Model) applyFilter() {
	filter, err := parseSessionFilter(m.filterQuery)
	if err != nil {
		filter = sessionFilter{} // shouldn't happen: submitFilter already validated it
	}
	items := make([]list.Item, 0, len(m.allSessions))
	for _, s := range m.allSessions {
		if filter.matches(s) {
			items = append(items, sessionItem{sess: s, marked: m.marked[s.ID]})
		}
	}
	m.sessionList.SetItems(items)
}

// beginFilter opens the advanced-filter prompt, pre-filled with the
// currently active query so it can be tweaked rather than retyped.
func (m *Model) beginFilter() {
	ti := textinput.New()
	ti.Placeholder = "days>7 kubernetes — empty clears"
	ti.SetValue(m.filterQuery)
	ti.CursorEnd()
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60
	m.filterInput = ti
	m.filterError = ""
	m.returnTo = m.state
	m.state = viewFilter
}

// submitFilter validates the query typed into m.filterInput and, if valid,
// applies it and returns to the session list.
func (m *Model) submitFilter() {
	query := strings.TrimSpace(m.filterInput.Value())
	if _, err := parseSessionFilter(query); err != nil {
		m.filterError = err.Error()
		return
	}
	m.filterQuery = query
	m.applyFilter()
	if query == "" {
		m.status = "Filter cleared."
	} else {
		m.status = fmt.Sprintf("Filter applied: %d of %d session(s) shown.", len(m.sessionList.Items()), len(m.allSessions))
	}
	m.state = m.returnTo
}

// reloadAgents rescans every provider from scratch — including re-checking
// Detect(), so an agent installed (or a path fixed) since the last scan
// shows up without restarting triAGe.
func (m *Model) reloadAgents() {
	m.agentList.SetItems(buildAgentItems(session.All(m.cfg.AgentPaths)))
}

// beginSetPath opens the path-input prompt for the highlighted agent,
// pre-filled with its current (possibly overridden) config directory so the
// user can tweak it rather than retype it from scratch.
func (m *Model) beginSetPath() {
	it, ok := m.agentList.SelectedItem().(agentItem)
	if !ok {
		return
	}
	m.pathTargetID = it.provider.ID()
	m.pathTargetName = it.provider.Name()
	m.pathError = ""

	ti := textinput.New()
	ti.Placeholder = "e.g. ~/.claude or /custom/path — empty clears the override"
	ti.SetValue(m.cfg.AgentPaths[m.pathTargetID])
	ti.CursorEnd()
	ti.Focus()
	ti.CharLimit = 4096
	ti.Width = 60
	m.pathInput = ti

	m.returnTo = m.state
	m.state = viewSetPath
}

// submitSetPath validates and saves the path currently typed into
// m.pathInput, or — if it's empty — clears any existing override so the
// agent falls back to its built-in default.
func (m *Model) submitSetPath() {
	raw := strings.TrimSpace(m.pathInput.Value())

	if raw == "" {
		delete(m.cfg.AgentPaths, m.pathTargetID)
	} else {
		path := expandHome(raw)
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		fi, err := os.Stat(path)
		if err != nil {
			m.pathError = "can't find " + path
			return
		}
		if !fi.IsDir() {
			m.pathError = path + " isn't a directory"
			return
		}
		if m.cfg.AgentPaths == nil {
			m.cfg.AgentPaths = map[string]string{}
		}
		m.cfg.AgentPaths[m.pathTargetID] = path
	}

	if err := m.cfg.Save(); err != nil {
		m.pathError = "couldn't save config: " + err.Error()
		return
	}

	m.reloadAgents()
	m.status = "Updated config path for " + m.pathTargetName + "."
	m.state = m.returnTo
}

// expandHome expands a leading "~" the way a shell would, since textinput
// doesn't do this itself and it's the natural way to type a home-relative
// path.
func expandHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// beginResume hands the terminal over to the active agent's own CLI so the
// user can continue the highlighted session interactively, via Bubble Tea's
// tea.ExecProcess (the standard way a Bubble Tea program suspends itself,
// runs an external interactive program with inherited stdio, and resumes
// once it exits — the same mechanism apps use to shell out to $EDITOR).
func (m *Model) beginResume() tea.Cmd {
	si, ok := m.sessionList.SelectedItem().(sessionItem)
	if !ok || m.active == nil {
		return nil
	}

	resumer, ok := m.active.(session.Resumable)
	if !ok {
		m.status = errorStyle.Render(m.active.Name() + " sessions can't be resumed from triAGe yet.")
		return nil
	}

	argv, dir, err := resumer.ResumeCommand(si.sess)
	if err != nil {
		m.status = errorStyle.Render("Can't resume: " + err.Error())
		return nil
	}
	if _, err := exec.LookPath(argv[0]); err != nil {
		m.status = errorStyle.Render(fmt.Sprintf("%q isn't on your PATH: %v", argv[0], err))
		return nil
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	title := si.sess.Title
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return resumeFinishedMsg{sessionTitle: title, err: err}
	})
}

func (m *Model) toggleMark() tea.Cmd {
	it, ok := m.sessionList.SelectedItem().(sessionItem)
	if !ok {
		return nil
	}
	it.marked = !it.marked
	m.marked[it.sess.ID] = it.marked
	if !it.marked {
		delete(m.marked, it.sess.ID)
	}
	// GlobalIndex(), not Index(): Index() is the cursor position within the
	// filtered subset while a filter is applied, but SetItem() expects an
	// index into the underlying unfiltered list.
	return m.sessionList.SetItem(m.sessionList.GlobalIndex(), it)
}

// beginDelete opens a confirmation dialog for either every marked session,
// or the highlighted one if nothing is marked. Marks are checked against
// every session for the active agent, not just what a filter currently
// shows, so marking something and then filtering it out of view doesn't
// silently drop it from the delete.
func (m *Model) beginDelete() {
	var targets []session.Session
	for _, s := range m.allSessions {
		if m.marked[s.ID] {
			targets = append(targets, s)
		}
	}
	if len(targets) == 0 {
		if si, ok := m.sessionList.SelectedItem().(sessionItem); ok {
			targets = []session.Session{si.sess}
		}
	}
	if len(targets) == 0 {
		return
	}
	m.confirm = &confirmTarget{sessions: targets}
	m.returnTo = viewSessions
	m.state = viewConfirm
}

func (m *Model) performDelete() {
	if m.confirm == nil || m.active == nil {
		return
	}
	var failed int
	for _, s := range m.confirm.sessions {
		if err := m.active.DeleteSession(s); err != nil {
			failed++
		}
		delete(m.marked, s.ID)
	}
	n := len(m.confirm.sessions)
	if failed == 0 {
		m.status = fmt.Sprintf("Deleted %d session(s).", n)
	} else {
		m.status = errorStyle.Render(fmt.Sprintf("Deleted %d session(s), %d failed.", n-failed, failed))
	}
	m.confirm = nil
	m.loadSessions(m.active)
	m.reloadAgents()
}

func (m Model) View() string {
	switch m.state {
	case viewConfirm:
		return m.viewConfirmDialog()
	case viewSetPath:
		return m.viewSetPathDialog()
	case viewFilter:
		return m.viewFilterDialog()
	case viewSessions:
		return m.viewSessions()
	default:
		return m.viewAgents()
	}
}

func (m Model) viewAgents() string {
	title := brandBar("AI agents on " + hostname)
	body := m.agentList.View()
	help := helpStyle.Render("↑/↓ move · enter open · p set path · r refresh · / filter · q quit")
	status := statusBarStyle.Width(max(m.width, 1)).Render(m.status)
	return lipgloss.JoinVertical(lipgloss.Left, title, body, help, status)
}

func (m Model) viewSessions() string {
	name := "Sessions"
	if m.active != nil {
		name = m.active.Name() + " sessions"
	}
	if m.filterQuery != "" {
		name += " — filter: " + m.filterQuery
	}
	title := brandBar(name)
	body := m.sessionList.View()
	help := helpStyle.Render("↑/↓ move · enter resume · space mark · d delete · f filter · esc back · r refresh · / search · q quit")
	status := statusBarStyle.Width(max(m.width, 1)).Render(m.status)
	return lipgloss.JoinVertical(lipgloss.Left, title, body, help, status)
}

func (m Model) viewConfirmDialog() string {
	if m.confirm == nil {
		return ""
	}
	n := len(m.confirm.sessions)
	var body string
	if n == 1 {
		s := m.confirm.sessions[0]
		body = fmt.Sprintf("Delete this session?\n\n  %s\n  %s\n\nThis permanently removes it from disk and cannot be undone.",
			s.Title, s.Project)
	} else {
		body = fmt.Sprintf("Delete %d marked sessions?\n\nThis permanently removes them from disk and cannot be undone.", n)
	}

	dialog := dialogBorderStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			dialogTitleStyle.Render("Confirm delete"),
			"",
			body,
			"",
			helpStyle.Render("y/enter confirm · n/esc cancel"),
		),
	)
	return lipgloss.Place(max(m.width, 1), max(m.height, 1), lipgloss.Center, lipgloss.Center, dialog)
}

func (m Model) viewSetPathDialog() string {
	lines := []string{
		promptTitleStyle.Render("Set config path — " + m.pathTargetName),
		"",
		m.pathInput.View(),
	}
	if m.pathError != "" {
		lines = append(lines, "", errorStyle.Render(m.pathError))
	}
	lines = append(lines, "", helpStyle.Render("enter save · esc cancel"))

	dialog := promptBorderStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return lipgloss.Place(max(m.width, 1), max(m.height, 1), lipgloss.Center, lipgloss.Center, dialog)
}

func (m Model) viewFilterDialog() string {
	lines := []string{
		promptTitleStyle.Render("Filter sessions"),
		"",
		m.filterInput.View(),
		"",
		helpStyle.Render("days>N / days<N for age, plus an optional regex — e.g. \"days>7 kube\""),
	}
	if m.filterError != "" {
		lines = append(lines, "", errorStyle.Render(m.filterError))
	}
	lines = append(lines, "", helpStyle.Render("enter apply · esc cancel"))

	dialog := promptBorderStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return lipgloss.Place(max(m.width, 1), max(m.height, 1), lipgloss.Center, lipgloss.Center, dialog)
}
