package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorBG        = lipgloss.Color("#1C2E27")
	colorSurface   = lipgloss.Color("#273A32")
	colorFG        = lipgloss.Color("#FEFAE0")
	colorSubtext   = lipgloss.Color("#E3D8AD")
	colorAccent    = lipgloss.Color("#DDA15E")
	colorWarning   = lipgloss.Color("#DDA15E")
	colorError     = lipgloss.Color("#BC6C25")
	colorSuccess   = lipgloss.Color("#7E8C4A")
	colorFolder    = lipgloss.Color("#4E5E2D")
	colorBorder    = lipgloss.Color("#465428")
	colorHighlight = lipgloss.Color("#523D2B")
)

var (
	baseTextStyle = lipgloss.NewStyle().Background(colorBG)
	appStyle      = lipgloss.NewStyle().
			Background(colorBG).
			Foreground(colorFG).
			Padding(1, 2)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			Background(colorBG)
	accentPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1).
				Background(colorBG)
	headerTextStyle = baseTextStyle.Copy().Foreground(colorFG).Bold(true)
	accentTextStyle = baseTextStyle.Copy().Foreground(colorAccent).Bold(true)
	mutedTextStyle  = baseTextStyle.Copy().Foreground(colorSubtext)
	successText     = baseTextStyle.Copy().Foreground(colorSuccess).Bold(true)
	warningText     = baseTextStyle.Copy().Foreground(colorWarning).Bold(true)
	errorText       = baseTextStyle.Copy().Foreground(colorError).Bold(true)
	folderText      = baseTextStyle.Copy().Foreground(colorFolder).Bold(true)
	selectedRow     = lipgloss.NewStyle().Background(colorHighlight).Foreground(colorFG)
)
