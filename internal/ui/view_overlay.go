package ui

import (
	"fmt"
	"strings"

	"github.com/NiladriHazra/filerepo/internal/download"
	gh "github.com/NiladriHazra/filerepo/internal/github"
	"github.com/charmbracelet/lipgloss"
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

	rendered := renderPreviewLines(lines[start:end], start, m.preview.showNumbers, m.preview.wrap, panelWidth)
	titleLine := accentTextStyle.Render(" "+title+" ") + mutedTextStyle.Render(" ["+itemTypeLabel(gh.RepoItem{Name: m.preview.path, ItemType: "file"})+"]")

	return titledPanelWithColor(
		titleLine,
		strings.Join([]string{
			rendered,
			"",
			joinShortcuts(
				shortcutLabel("Esc", "close"),
				shortcutLabel("j/k", "scroll"),
				shortcutLabel("w", "wrap"),
				shortcutLabel("n", "numbers"),
			),
		}, "\n"),
		panelWidth,
		colorAccent,
	)
}

func renderPreviewLines(lines []string, lineOffset int, showNumbers, wrap bool, panelWidth int) string {
	contentWidth := max(24, panelWidth-8)
	numberWidth := 0
	if showNumbers {
		numberWidth = len(fmt.Sprintf("%d", len(lines)))
	}

	codeStyle := baseTextStyle.Copy().Foreground(colorFG)
	rendered := make([]string, 0, len(lines))
	for index, line := range lines {
		current := strings.ReplaceAll(line, "\t", "    ")
		if !wrap && lipgloss.Width(current) > contentWidth-numberWidth-2 {
			current = truncate(current, contentWidth-numberWidth-2)
		}
		current = codeStyle.Render(current)

		if showNumbers {
			prefix := mutedTextStyle.Render(fmt.Sprintf("%*d ", numberWidth, index+1+lineOffset))
			current = prefix + current
		}
		rendered = append(rendered, current)
	}

	return strings.Join(rendered, "\n")
}

func (m *model) renderRefPicker() string {
	title := accentTextStyle.Render(" Switch Ref ")
	refs := m.filteredRefs()
	if len(refs) == 0 {
		return titledPanelWithColor(title, mutedTextStyle.Render("No matching refs."), max(52, min(fullPanelWidth(m.width), 90)), colorAccent)
	}

	lines := []string{
		mutedTextStyle.Render("Filter: ") + nonEmpty(m.refPicker.query, "_"),
		"",
	}
	start := max(m.refPicker.cursor-6, 0)
	end := min(start+12, len(refs))
	for index := start; index < end; index++ {
		ref := refs[index]
		line := fmt.Sprintf("%-7s %s", strings.ToUpper(ref.Kind), ref.Name)
		if ref.IsDefault {
			line += "  (default)"
		}
		if index == m.refPicker.cursor {
			lines = append(lines, selectedRow.Render(line))
			continue
		}
		lines = append(lines, line)
	}

	lines = append(lines, "", joinShortcuts(
		shortcutLabel("Enter", "switch"),
		shortcutLabel("Esc", "close"),
		shortcutLabel("type", "filter"),
	))

	return panelWithColor(strings.Join(lines, "\n"), max(52, min(fullPanelWidth(m.width), 90)), colorAccent)
}

func (m *model) renderInfoOverlay() string {
	tabNames := []string{"Summary", "README", "Releases"}
	title := accentTextStyle.Render(" Repository ")
	var body string

	switch m.infoOverlay.tab {
	case 0:
		body = m.renderRepoSummary()
	case 1:
		body = m.renderRepoReadme()
	case 2:
		body = m.renderRepoReleases()
	}

	header := strings.Join([]string{
		tabLabel(tabNames[0], m.infoOverlay.tab == 0),
		tabLabel(tabNames[1], m.infoOverlay.tab == 1),
		tabLabel(tabNames[2], m.infoOverlay.tab == 2),
	}, "  ")

	return titledPanelWithColor(title, strings.Join([]string{
		header,
		"",
		body,
		"",
		joinShortcuts(
			shortcutLabel("Tab", "next"),
			shortcutLabel("j/k", "scroll"),
			shortcutLabel("Esc", "close"),
		),
	}, "\n"), max(68, min(fullPanelWidth(m.width), 132)), colorAccent)
}

func (m *model) renderReleasePicker() string {
	assets := m.releaseItems()
	title := accentTextStyle.Render(" Release Assets ")
	if len(assets) == 0 {
		return titledPanelWithColor(title, mutedTextStyle.Render("No release assets available."), max(56, min(fullPanelWidth(m.width), 100)), colorAccent)
	}

	lines := []string{}
	start := max(m.releasePicker.cursor-6, 0)
	end := min(start+12, len(assets))
	for index := start; index < end; index++ {
		asset := assets[index]
		line := fmt.Sprintf("%-20s  %-10s  %s", asset.Status, humanSize(asset.ActualSize()), truncate(asset.Name, 48))
		if index == m.releasePicker.cursor {
			lines = append(lines, selectedRow.Render(line))
			continue
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", joinShortcuts(
		shortcutLabel("Enter", "download"),
		shortcutLabel("y", "copy url"),
		shortcutLabel("Esc", "close"),
	))

	return titledPanelWithColor(title, strings.Join(lines, "\n"), max(68, min(fullPanelWidth(m.width), 110)), colorAccent)
}

func (m *model) renderRepoSummary() string {
	lines := []string{
		headerTextStyle.Render(nonEmpty(m.repoMetadata.FullName, m.currentRepoURL)),
	}
	if m.repoMetadata.Description != "" {
		lines = append(lines, mutedTextStyle.Render(m.repoMetadata.Description))
	}
	lines = append(lines,
		fmt.Sprintf("Default Branch: %s", nonEmpty(m.repoMetadata.DefaultBranch, m.currentURL.Branch)),
		fmt.Sprintf("Stars: %d  Forks: %d  Open Issues: %d", m.repoMetadata.Stars, m.repoMetadata.Forks, m.repoMetadata.OpenIssues),
		fmt.Sprintf("Language: %s  Visibility: %s", nonEmpty(m.repoMetadata.Language, "n/a"), visibilityLabel(m.repoMetadata.Private)),
		fmt.Sprintf("Auth Status: %s", m.authStatusLabel()),
	)
	if !m.repoMetadata.UpdatedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("Updated: %s", m.repoMetadata.UpdatedAt.Local().Format("2006-01-02 15:04")))
	}
	if !m.repoMetadata.PushedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("Pushed:  %s", m.repoMetadata.PushedAt.Local().Format("2006-01-02 15:04")))
	}
	if len(m.repoRefs) > 0 {
		lines = append(lines, "", successText.Render("Refs"))
		for _, ref := range m.repoRefs[:min(len(m.repoRefs), 8)] {
			lines = append(lines, fmt.Sprintf("  %-7s %s", strings.ToUpper(ref.Kind), ref.Name))
		}
	}
	return scrollLines(lines, m.infoOverlay.scroll, m.height-10)
}

func (m *model) renderRepoReadme() string {
	body := m.repoReadme.Content
	if stringsTrimmed(body) == "" {
		body = "No README available."
	}
	lines := strings.Split(body, "\n")
	return scrollLines(lines, m.infoOverlay.scroll, m.height-10)
}

func (m *model) renderRepoReleases() string {
	if len(m.repoReleases) == 0 {
		return "No releases found."
	}

	lines := make([]string, 0, len(m.repoReleases)*3)
	for _, release := range m.repoReleases[:min(len(m.repoReleases), 6)] {
		name := nonEmpty(release.Name, release.TagName)
		lines = append(lines, headerTextStyle.Render(name))
		lines = append(lines, fmt.Sprintf("  Tag: %s  Assets: %d  Published: %s", release.TagName, len(release.Assets), release.PublishedAt.Local().Format("2006-01-02")))
		if release.Body != "" {
			lines = append(lines, "  "+truncate(strings.ReplaceAll(release.Body, "\n", " "), 96))
		}
		lines = append(lines, "")
	}
	return scrollLines(lines, m.infoOverlay.scroll, m.height-10)
}

func scrollLines(lines []string, offset, height int) string {
	if len(lines) == 0 {
		return ""
	}
	start := min(offset, max(len(lines)-1, 0))
	end := min(start+max(height, 4), len(lines))
	return strings.Join(lines[start:end], "\n")
}

func tabLabel(value string, active bool) string {
	if active {
		return successText.Render("[" + value + "]")
	}
	return mutedTextStyle.Render(value)
}

func visibilityLabel(private bool) string {
	if private {
		return "private"
	}
	return "public"
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
		fmt.Sprintf("Conflict: %s", formatConflictStrategy(m.savePrompt.conflict)),
		fmt.Sprintf("Output:   %s", formatOutputMode(m.savePrompt.output)),
		"",
		joinShortcuts(
			shortcutLabel("Enter", "confirm"),
			shortcutLabel("Esc", "cancel"),
			shortcutLabel("s/o/r/e", "conflict"),
			shortcutLabel("f/z/t", "output"),
		),
	}

	return panelWithColor(strings.Join(body, "\n"), max(52, min(fullPanelWidth(m.width), 96)), colorBorder)
}

func formatConflictStrategy(strategy download.ConflictStrategy) string {
	switch strategy {
	case download.ConflictOverwrite:
		return "overwrite"
	case download.ConflictRename:
		return "rename"
	case download.ConflictResume:
		return "resume"
	default:
		return "skip"
	}
}

func formatOutputMode(mode download.OutputMode) string {
	switch mode {
	case download.OutputZip:
		return "zip archive"
	case download.OutputTarGz:
		return "tar.gz archive"
	default:
		return "regular files"
	}
}
