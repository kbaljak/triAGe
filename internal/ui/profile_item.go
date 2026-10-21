package ui

import (
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kbaljak/triAGe/internal/session"
)

// profileItem is one row in an agent's profile list: a named config
// directory for that agent (always including "Default", the implicit
// profile backed by the agent's built-in path or its single-path override).
type profileItem struct {
	provider  session.Provider
	name      string
	isDefault bool // the built-in profile — can't be deleted
	detected  bool
}

func (i profileItem) FilterValue() string { return i.name }

// profileDelegate renders profileItem rows: name, detected status, and the
// resolved directory it reads from. Deliberately no session count/size here
// — that would mean a full session scan per profile just to render the
// picker; the real count shows up once a profile is actually opened.
type profileDelegate struct{}

func (d profileDelegate) Height() int                             { return 2 }
func (d profileDelegate) Spacing() int                            { return 1 }
func (d profileDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d profileDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(profileItem)
	if !ok {
		return
	}
	selected := index == m.Index()

	nameStyle := lipgloss.NewStyle().Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(colorSubtle)
	if !it.detected {
		nameStyle = notInstalledStyle
	}
	if selected {
		nameStyle = nameStyle.Foreground(colorAccent)
	}

	cursor := "  "
	if selected {
		cursor = lipgloss.NewStyle().Foreground(colorAccent).Render("▸ ")
	}

	name := it.name
	if it.isDefault {
		name += " (default)"
	}

	line2 := "    "
	if !it.detected {
		line2 += notInstalledStyle.Render("not detected") + metaStyle.Render(" · ")
	}
	line2 += metaStyle.Render(it.provider.Home())

	line1 := cursor + nameStyle.Render(name)
	io.WriteString(w, strings.Join([]string{line1, line2}, "\n"))
}
