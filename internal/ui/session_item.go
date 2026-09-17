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

// sessionItem is one row in the per-agent session list.
type sessionItem struct {
	sess   session.Session
	marked bool
}

func (i sessionItem) FilterValue() string { return i.sess.Title + " " + i.sess.Project }

// sessionDelegate renders sessionItem rows: a mark checkbox, title, and a
// second line with project path, recency and size.
type sessionDelegate struct{}

func (d sessionDelegate) Height() int                             { return 2 }
func (d sessionDelegate) Spacing() int                            { return 1 }
func (d sessionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(sessionItem)
	if !ok {
		return
	}
	selected := index == m.Index()

	titleStyle := lipgloss.NewStyle().Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(colorSubtle)
	if selected {
		titleStyle = titleStyle.Foreground(colorAccent)
	}
	if it.marked {
		titleStyle = titleStyle.Foreground(colorWarn)
	}

	cursor := "  "
	if selected {
		cursor = lipgloss.NewStyle().Foreground(colorAccent).Render("▸ ")
	}

	checkbox := "[ ]"
	if it.marked {
		checkbox = lipgloss.NewStyle().Foreground(colorWarn).Render("[x]")
	}

	title := it.sess.Title
	if title == "" {
		title = "(untitled)"
	}

	line1 := cursor + checkbox + " " + titleStyle.Render(title)
	line2 := fmt.Sprintf("      %s · %s · %s",
		it.sess.Project,
		formatRelativeTime(it.sess.UpdatedAt),
		formatBytes(it.sess.SizeBytes),
	)
	fmt.Fprint(w, strings.Join([]string{line1, metaStyle.Render(line2)}, "\n"))
}
