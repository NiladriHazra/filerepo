package ui

import (
	"strings"
)

func (m *model) View() string {
	var body string

	switch {
	case m.savePrompt != nil:
		body = m.renderSavePrompt()
	case m.preview != nil:
		body = m.renderPreview()
	default:
		switch m.mode {
		case modeInput:
			body = m.renderInput()
		case modeLoading:
			body = m.renderLoading()
		case modeBrowse:
			body = m.renderBrowser()
		}
	}

	if m.toast != nil {
		body += "\n\n" + m.renderToast()
	}

	return appStyle.Render(body)
}

func (m *model) renderInput() string {
	inputValue := m.urlInput
	switch {
	case inputValue == "":
		inputValue = "_ Press Tab to auto-fill GitHub URL"
	case m.urlCursor >= len(inputValue):
		inputValue += "_"
	default:
		inputValue = inputValue[:m.urlCursor] + "_" + inputValue[m.urlCursor+1:]
	}

	sections := []string{
		renderLogo(),
		mutedTextStyle.Render("Grab any file or folder from GitHub. No clones. Just what you need."),
		accentPanelStyle.Render("GitHub URL\n" + inputValue),
		panelStyle.Render(strings.Join([]string{
			successText.Render("Examples"),
			"  https://github.com/torvalds/linux",
			"  https://github.com/rust-lang/rust/tree/master/src/tools",
			"  https://github.com/user/repo/tree/main/some-folder",
			"",
			mutedTextStyle.Render("Enter start  |  Tab auto-fill  |  Esc quit"),
		}, "\n")),
	}

	return strings.Join(sections, "\n\n")
}

func renderLogo() string {
	lines := []string{
		"███████╗██╗██╗     ███████╗██████╗ ███████╗██████╗  ██████╗ ",
		"██╔════╝██║██║     ██╔════╝██╔══██╗██╔════╝██╔══██╗██╔═══██╗",
		"█████╗  ██║██║     █████╗  ██████╔╝█████╗  ██████╔╝██║   ██║",
		"██╔══╝  ██║██║     ██╔══╝  ██╔══██╗██╔══╝  ██╔═══╝ ██║   ██║",
		"██║     ██║███████╗███████╗██║  ██║███████╗██║     ╚██████╔╝",
		"╚═╝     ╚═╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝╚═╝      ╚═════╝ ",
	}

	for index := range lines {
		lines[index] = accentTextStyle.Render(lines[index])
	}

	return strings.Join(lines, "\n")
}

func (m *model) renderToast() string {
	style := mutedTextStyle
	switch m.toast.kind {
	case toastSuccess:
		style = successText
	case toastWarning:
		style = warningText
	case toastError:
		style = errorText
	case toastInfo:
		style = accentTextStyle
	}

	return panelStyle.Width(max(44, min(m.width-6, 96))).Render(style.Render(m.toast.message))
}
