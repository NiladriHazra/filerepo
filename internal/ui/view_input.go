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
	case m.refPicker != nil:
		body = m.renderRefPicker()
	case m.releasePicker != nil:
		body = m.renderReleasePicker()
	case m.infoOverlay != nil:
		body = m.renderInfoOverlay()
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

	return appStyle.
		Width(m.width).
		Height(m.height).
		Render(body)
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

	contentWidth := boundedContentWidth(m.width, inputPanelMinWidth, inputPanelMaxWidth, inputPanelGutter)

	footer := joinShortcuts(
		shortcutLabel("Enter", "start"),
		shortcutLabel("Tab", "auto-fill"),
		shortcutLabel("Esc", "quit"),
	)

	sections := []string{
		centerBlock(contentWidth, renderLogo()),
		centerBlock(contentWidth, mutedTextStyle.Render("Grab any file or folder from GitHub. No clones. Just what you need.")),
		titledPanelWithColor(
			accentTextStyle.Render(" GitHub URL "),
			inputValue,
			contentWidth,
			colorAccent,
		),
		titledPanelWithColor(successText.Render(" Examples "), strings.Join([]string{
			"    https://github.com/torvalds/linux",
			"    https://github.com/rust-lang/rust/tree/master/src/tools",
			"    https://github.com/user/repo/tree/main/some-folder",
			"",
			mutedTextStyle.Render("Tip: Private repo? use --token <TOKEN>"),
			mutedTextStyle.Render("     or save one with: filerepo config set token YOUR_TOKEN"),
		}, "\n"), contentWidth, colorBorder),
		m.renderRecentAndFavorites(contentWidth),
		centerBlock(contentWidth, footer),
	}

	return centerInputLayout(m.width, m.height, fillBlockBackground(strings.Join(sections, "\n\n")))
}

func (m *model) renderRecentAndFavorites(width int) string {
	lines := []string{
		mutedTextStyle.Render("Profile: " + nonEmpty(m.activeProfile, "default")),
	}

	if len(m.configState.Favorites) > 0 {
		lines = append(lines, "")
		lines = append(lines, successText.Render("Favorites:"))
		for _, entry := range m.configState.Favorites[:min(len(m.configState.Favorites), 3)] {
			lines = append(lines, "  "+entry.URL)
		}
	}

	if len(m.configState.RecentRepos) > 0 {
		lines = append(lines, "")
		lines = append(lines, mutedTextStyle.Render("Recent:"))
		for _, entry := range m.configState.RecentRepos[:min(len(m.configState.RecentRepos), 3)] {
			lines = append(lines, "  "+entry.URL)
		}
	}

	return titledPanelWithColor(successText.Render(" Saved "), strings.Join(lines, "\n"), width, colorBorder)
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
		lines[index] = accentTextStyle.Render(strings.TrimRight(lines[index], " "))
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

	return panelWithColor(style.Render(m.toast.message), max(44, min(m.width-6, 96)), colorBorder)
}
