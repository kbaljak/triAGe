// Package ui implements triAGe's terminal GUI: a two-screen browser over
// every AI agent provider in internal/session, letting the user drill into
// an agent's sessions to resume, mark, and delete them.
package ui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
)

// confirmTarget describes what a pending delete confirmation would delete.
type confirmTarget struct {
	sessions []session.Session
}

// Model is the root Bubble Tea model for triAGe.
type Model struct {
	state    viewState
	returnTo viewState // state to go back to after a confirm dialog

	agentList   list.Model
	sessionList list.Model

	active session.Provider // provider currently open in the session view
	marked map[string]bool  // session ID -> marked, scoped to the active provider

	confirm *confirmTarget
	status  string // transient status line, replaced on the next action

	width, height int
}

// New builds the initial model, eagerly scanning every detected provider so
// the agent list can show session counts right away.
func New() Model {
	var items []list.Item
	for _, p := range session.All() {
		if !p.Detect() {
			continue
		}
		sessions, err := p.ListSessions()
		var total int64
		for _, s := range sessions {
			total += s.SizeBytes
		}
		items = append(items, agentItem{
			provider:     p,
			sessionCount: len(sessions),
			totalSize:    total,
			loadErr:      err,
		})
	}

	agentList := list.New(items, agentDelegate{}, 0, 0)
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

	return Model{
		state:       viewAgents,
		agentList:   agentList,
		sessionList: sessionList,
		marked:      map[string]bool{},
	}
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
		}
		return m, nil
	}

	// Anything else (the list's async filter-match results, the filter
	// input's cursor blink, etc.) still needs to reach whichever list is
	// active, or filtering silently never finishes applying.
	var cmd tea.Cmd
	switch m.state {
	case viewAgents:
		m.agentList, cmd = m.agentList.Update(msg)
	case viewSessions:
		m.sessionList, cmd = m.sessionList.Update(msg)
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
		if it, ok := m.agentList.SelectedItem().(agentItem); ok {
			m.openAgent(it.provider)
		}
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

// openAgent switches into the session view for the given provider, loading
// (or reloading) its sessions and clearing any stale marks.
func (m *Model) openAgent(p session.Provider) {
	m.active = p
	m.marked = map[string]bool{}
	m.loadSessions(p)
	m.state = viewSessions
	m.status = ""
}

func (m *Model) loadSessions(p session.Provider) {
	sessions, err := p.ListSessions()
	items := make([]list.Item, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, sessionItem{sess: s, marked: m.marked[s.ID]})
	}
	m.sessionList.SetItems(items)
	if err != nil {
		m.status = errorStyle.Render("Failed to load sessions: " + err.Error())
	}
}

func (m *Model) reloadAgents() {
	items := m.agentList.Items()
	for i, it := range items {
		ai, ok := it.(agentItem)
		if !ok {
			continue
		}
		sessions, err := ai.provider.ListSessions()
		var total int64
		for _, s := range sessions {
			total += s.SizeBytes
		}
		ai.sessionCount = len(sessions)
		ai.totalSize = total
		ai.loadErr = err
		m.agentList.SetItem(i, ai)
	}
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
// or the highlighted one if nothing is marked.
func (m *Model) beginDelete() {
	var targets []session.Session
	for _, it := range m.sessionList.Items() {
		if si, ok := it.(sessionItem); ok && si.marked {
			targets = append(targets, si.sess)
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
	case viewSessions:
		return m.viewSessions()
	default:
		return m.viewAgents()
	}
}

func (m Model) viewAgents() string {
	title := brandBar("AI agents on " + hostname)
	body := m.agentList.View()
	help := helpStyle.Render("↑/↓ move · enter open · r refresh · / filter · q quit")
	status := statusBarStyle.Width(max(m.width, 1)).Render(m.status)
	return lipgloss.JoinVertical(lipgloss.Left, title, body, help, status)
}

func (m Model) viewSessions() string {
	name := "Sessions"
	if m.active != nil {
		name = m.active.Name() + " sessions"
	}
	title := brandBar(name)
	body := m.sessionList.View()
	help := helpStyle.Render("↑/↓ move · enter resume · space mark · d delete · esc back · r refresh · / filter · q quit")
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
