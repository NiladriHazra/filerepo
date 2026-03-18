package ui

import (
	"fmt"
	"strings"
)

func (m *model) renderPreview() string {
	title := "Preview"
	switch m.preview.status {
	case previewLoading:
		title = "Previewing " + m.preview.path
	case previewError:
		title = "Preview Unavailable"
	}

	body := m.preview.content
	switch m.preview.status {
	case previewLoading:
		body = "Loading file preview..."
	case previewError:
		body = m.preview.content
	}
	if stringsTrimmed(body) == "" {
		body = "(empty file)"
	}

	lines := strings.Split(body, "\n")
	start := min(m.preview.scroll, max(len(lines)-1, 0))
	end := min(start+m.height-8, len(lines))
	panelWidth := fullPanelWidth(m.width)

	return titledPanelWithColor(
		accentTextStyle.Render(" "+title+" "),
		strings.Join([]string{
			strings.Join(lines[start:end], "\n"),
			"",
			joinShortcuts(
				shortcutLabel("Esc", "close"),
				shortcutLabel("j/k", "scroll"),
			),
		}, "\n"),
		panelWidth,
		colorAccent,
	)
}

func (m *model) renderSavePrompt() string {
	input := m.savePrompt.input
	switch {
	case m.savePrompt.cursor >= len(input):
		input += "_"
	default:
		input = input[:m.savePrompt.cursor] + "_" + input[m.savePrompt.cursor+1:]
	}

	body := []string{
		successText.Render(fmt.Sprintf("%d item(s)", m.savePrompt.itemCount)) + mutedTextStyle.Render(" will be saved into this directory."),
		"",
		titledPanelWithColor(accentTextStyle.Render(" Directory Path "), input, max(40, min(m.width-12, 88)), colorAccent),
		"",
		joinShortcuts(
			shortcutLabel("Enter", "confirm"),
			shortcutLabel("Esc", "cancel"),
		),
	}

	return panelWithColor(strings.Join(body, "\n"), max(52, min(fullPanelWidth(m.width), 96)), colorBorder)
}
