package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent = lipgloss.Color("212") // pink — selection / accent
	colorSubtle = lipgloss.Color("245") // grey — secondary text
	colorFaint  = lipgloss.Color("244") // dim grey — disabled / "no sessions" text
	colorHelp   = lipgloss.Color("255") // bright, near-white — command hints (needs to read on translucent terminals)
	colorDanger = lipgloss.Color("203") // red — delete / destructive
	colorGood   = lipgloss.Color("114") // green — installed / ok
	colorWarn   = lipgloss.Color("221") // yellow — marked items

	titleBg = lipgloss.Color("62") // title bar background, shared by every segment of brandBar

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(titleBg)
	titleAGStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Background(titleBg)

	helpStyle = lipgloss.NewStyle().Foreground(colorHelp).Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)

	dialogBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorDanger).
				Padding(1, 3)

	dialogTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorDanger)

	promptBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(1, 3)

	promptTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	notInstalledStyle = lipgloss.NewStyle().Foreground(colorFaint).Italic(true)
)

// brandBar renders the title bar as "triAGe — <suffix>", with the "AG" in
// "triAGe" (tri-AG-e, for "agent") picked out in the accent color. Each
// segment declares the same background explicitly rather than relying on
// titleStyle's Padding, since a nested style's trailing reset code would
// otherwise cut the background short partway through the line.
func brandBar(suffix string) string {
	return titleStyle.Render(" tri") + titleAGStyle.Render("AG") + titleStyle.Render("e — "+suffix+" ")
}
