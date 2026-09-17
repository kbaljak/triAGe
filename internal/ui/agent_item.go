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

// agentItem is one row in the top-level agent list.
type agentItem struct {
	provider     session.Provider
	sessionCount int
	totalSize    int64
	loadErr      error
}

func (i agentItem) FilterValue() string { return i.provider.Name() }

// agentDelegate renders agentItem rows: name, session count and total size.
type agentDelegate struct{}

func (d agentDelegate) Height() int                             { return 2 }
func (d agentDelegate) Spacing() int                            { return 1 }
func (d agentDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d agentDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(agentItem)
	if !ok {
		return
	}
	selected := index == m.Index()

	nameStyle := lipgloss.NewStyle().Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(colorSubtle)
	if selected {
		nameStyle = nameStyle.Foreground(colorAccent)
	}

	cursor := "  "
	if selected {
		cursor = lipgloss.NewStyle().Foreground(colorAccent).Render("▸ ")
	}

	var meta string
	switch {
	case it.loadErr != nil:
		meta = errorStyle.Render("error: " + it.loadErr.Error())
	case it.sessionCount == 0:
		meta = notInstalledStyle.Render("no sessions found")
	default:
		plural := "s"
		if it.sessionCount == 1 {
			plural = ""
		}
		meta = metaStyle.Render(fmt.Sprintf("%d session%s · %s on disk", it.sessionCount, plural, formatBytes(it.totalSize)))
	}

	line1 := cursor + nameStyle.Render(it.provider.Name())
	line2 := "    " + meta
	fmt.Fprint(w, strings.Join([]string{line1, line2}, "\n"))
}
