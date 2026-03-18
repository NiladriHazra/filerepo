package ui

import "strings"

func (m *model) renderLoading() string {
	frames := []string{"   .      ", "   ..     ", "   ...    ", "   ....   ", "    ...   ", "     ..   ", "      .   ", "          "}
	frame := frames[m.frame%len(frames)]

	return strings.Join([]string{
		renderLogo(),
		mutedTextStyle.Render(nonEmpty(m.statusMessage, "Loading repository...")),
		accentPanelStyle.Render(frame),
		mutedTextStyle.Render("Press Esc to cancel"),
	}, "\n\n")
}

func nonEmpty(value, fallback string) string {
	if stringsTrimmed(value) == "" {
		return fallback
	}
	return value
}
