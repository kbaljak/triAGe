// Command triage ("triAGe" — the AG is for "agent") is a terminal GUI for
// browsing, resuming, and cleaning up local session history left behind by
// AI coding agents (Claude Code, Antigravity, Gemini CLI, Codex, ...).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kbaljak/triAGe/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "triage: "+err.Error())
		os.Exit(1)
	}
}
