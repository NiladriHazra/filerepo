package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	inputPanelMaxWidth = 112
	inputPanelMinWidth = 72
	inputPanelGutter   = 10
	fullPanelMinWidth  = 60
	fullPanelMaxWidth  = 160
	fullPanelGutter    = 8
)

func appInnerSize(width, height int) (int, int) {
	innerWidth := max(width-appStyle.GetHorizontalFrameSize(), 1)
	innerHeight := max(height-appStyle.GetVerticalFrameSize(), 1)
	return innerWidth, innerHeight
}

func boundedContentWidth(totalWidth, minWidth, maxWidth, gutter int) int {
	innerWidth, _ := appInnerSize(totalWidth, 1)
	targetWidth := innerWidth - gutter

	switch {
	case targetWidth <= 0:
		return innerWidth
	case targetWidth > maxWidth:
		return maxWidth
	case innerWidth >= minWidth && targetWidth < minWidth:
		return minWidth
	default:
		return targetWidth
	}
}

func centerBlock(width int, content string) string {
	if width <= 0 {
		return content
	}
	return lipgloss.PlaceHorizontal(
		width,
		lipgloss.Center,
		content,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(colorBG),
	)
}

func inputTopPadding(totalHeight, contentHeight int) int {
	_, innerHeight := appInnerSize(1, totalHeight)
	if innerHeight <= contentHeight+4 {
		return 0
	}
	return max((innerHeight-contentHeight)/5, 1)
}

func centerInputLayout(width, height int, content string) string {
	innerWidth, _ := appInnerSize(width, height)
	paddingTop := inputTopPadding(height, lipgloss.Height(content))
	return strings.Repeat("\n", paddingTop) + centerBlock(innerWidth, content)
}

func centerLoadingLayout(width, height int, content string) string {
	innerWidth, innerHeight := appInnerSize(width, height)
	return lipgloss.Place(
		innerWidth,
		innerHeight,
		lipgloss.Center,
		lipgloss.Center,
		content,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(colorBG),
	)
}

func fillBlockBackground(content string) string {
	lines := strings.Split(content, "\n")
	width := 0
	for _, line := range lines {
		width = max(width, lipgloss.Width(line))
	}

	for index, line := range lines {
		lines[index] = baseTextStyle.Copy().Width(width).Render(line)
	}

	return strings.Join(lines, "\n")
}

func shortcutLabel(key, action string) string {
	return accentTextStyle.Render(key) + mutedTextStyle.Render(" "+action)
}

func joinShortcuts(items ...string) string {
	return strings.Join(items, mutedTextStyle.Render("  |  "))
}

func fullPanelWidth(totalWidth int) int {
	return boundedContentWidth(totalWidth, fullPanelMinWidth, fullPanelMaxWidth, fullPanelGutter)
}

func panelWithColor(content string, width int, borderColor lipgloss.TerminalColor) string {
	return titledPanelWithColor("", content, width, borderColor)
}

func titledPanelWithColor(title, content string, width int, borderColor lipgloss.TerminalColor) string {
	innerWidth := max(width, 1)
	borderStyle := lipgloss.NewStyle().Foreground(borderColor).Background(colorBG)
	lines := []string{buildPanelTopBorder(title, innerWidth, borderStyle)}
	lines = append(lines, renderPanelBody(content, innerWidth, borderStyle)...)
	lines = append(lines, borderStyle.Render("╰"+strings.Repeat("─", innerWidth+2)+"╯"))
	return strings.Join(lines, "\n")
}

func buildPanelTopBorder(title string, innerWidth int, borderStyle lipgloss.Style) string {
	if lipgloss.Width(title) == 0 {
		return borderStyle.Render("╭" + strings.Repeat("─", innerWidth+2) + "╮")
	}

	fillWidth := innerWidth + 1 - lipgloss.Width(title)
	if fillWidth < 0 {
		fillWidth = 0
	}

	return borderStyle.Render("╭─") + title + borderStyle.Render(strings.Repeat("─", fillWidth)+"╮")
}

func renderPanelBody(content string, width int, borderStyle lipgloss.Style) []string {
	sourceLines := strings.Split(content, "\n")
	if len(sourceLines) == 0 {
		sourceLines = []string{""}
	}

	lines := make([]string, 0, len(sourceLines))
	clampStyle := lipgloss.NewStyle().MaxWidth(width)
	fillStyle := baseTextStyle.Copy().Width(width)

	for _, sourceLine := range sourceLines {
		wrappedLines := strings.Split(clampStyle.Render(sourceLine), "\n")
		for _, wrappedLine := range wrappedLines {
			lines = append(
				lines,
				borderStyle.Render("│ ")+fillStyle.Render(wrappedLine)+borderStyle.Render(" │"),
			)
		}
	}

	return lines
}
