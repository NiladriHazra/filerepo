package ui

import "strings"

func (m *model) renderLoading() string {
	frames := []string{".", "..", "...", "....", "...", "..", "."}
	frame := frames[m.frame%len(frames)]
	contentWidth := boundedContentWidth(m.width, 28, 72, inputPanelGutter)

	sections := []string{
		centerBlock(contentWidth, renderLogo()),
		centerBlock(contentWidth, mutedTextStyle.Render(nonEmpty(m.statusMessage, "Loading repository..."))),
		centerBlock(contentWidth, accentTextStyle.Render(frame)),
		centerBlock(contentWidth, mutedTextStyle.Render("Press Esc to cancel")),
	}

	return centerLoadingLayout(m.width, m.height, fillBlockBackground(strings.Join(sections, "\n\n")))
}

func nonEmpty(value, fallback string) string {
	if stringsTrimmed(value) == "" {
		return fallback
	}
	return value
}
