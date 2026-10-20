package ui

import (
	"fmt"
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
	provider     session.Provider
	name         string
	isDefault    bool // the built-in profile — can't be deleted
	detected     bool
	sessionCount int
	totalSize    int64
	loadErr      error
}

func (i profileItem) FilterValue() string { return i.name }

// profileDelegate renders profileItem rows: name, session count/status, and
// the resolved directory it reads from.
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

	// Each case below renders its own complete, self-contained style span
	// (own start + reset) rather than nesting inside a shared outer style —
	// an inner reset code would otherwise cut an outer span short.
	var status string
	switch {
	case !it.detected:
		status = notInstalledStyle.Render("not detected")
	case it.loadErr != nil:
		status = errorStyle.Render("error: " + it.loadErr.Error())
	case it.sessionCount == 0:
		status = notInstalledStyle.Render("no sessions found")
	default:
		plural := "s"
		if it.sessionCount == 1 {
			plural = ""
		}
		status = metaStyle.Render(fmt.Sprintf("%d session%s · %s on disk", it.sessionCount, plural, formatBytes(it.totalSize)))
	}

	line1 := cursor + nameStyle.Render(name)
	line2 := "    " + status + metaStyle.Render(" · "+it.provider.Home())
	fmt.Fprint(w, strings.Join([]string{line1, line2}, "\n"))
}
