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

	return accentPanelStyle.Width(max(60, min(m.width-6, 120))).Render(
		title + "\n\n" +
			strings.Join(lines[start:end], "\n") + "\n\n" +
			mutedTextStyle.Render("Esc close  |  j/k scroll"),
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
		accentPanelStyle.Render("Directory Path\n" + input),
		"",
		mutedTextStyle.Render("Enter confirm  |  Esc cancel"),
	}

	return panelStyle.Width(max(52, min(m.width-8, 96))).Render(strings.Join(body, "\n"))
}
