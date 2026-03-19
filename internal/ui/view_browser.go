package ui

import (
	"fmt"
	"strings"

	gh "github.com/NiladriHazra/filerepo/internal/github"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderBrowser() string {
	panelWidth := fullPanelWidth(m.width)

	sections := []string{
		m.renderRepositoryHeader(panelWidth),
		m.renderFileList(panelWidth),
	}

	if m.downloading && m.downloadProgress != nil {
		sections = append(sections, m.renderDownloadProgress())
	}
	if m.searching {
		sections = append(sections, titledPanelWithColor(
			accentTextStyle.Render(" Search "),
			"/"+m.searchQuery+"_",
			panelWidth,
			colorAccent,
		))
	}

	sections = append(sections, joinShortcuts(
		shortcutLabel("j/k", "nav"),
		shortcutLabel("Enter", "open/preview"),
		shortcutLabel("Space", "select"),
		shortcutLabel("a/u", "all/none"),
		shortcutLabel("d", "download"),
		shortcutLabel("/", "search"),
		shortcutLabel("b/t", "refs"),
		shortcutLabel("R", "releases"),
		shortcutLabel("m", "repo"),
		shortcutLabel("y/Y/P", "copy"),
		shortcutLabel("f", "favorite"),
	))

	return strings.Join(sections, "\n\n")
}

func (m *model) renderRepositoryHeader(panelWidth int) string {
	info := ""
	if m.currentURL != nil {
		switch m.currentURL.Kind {
		case gh.TargetCompare:
			info = fmt.Sprintf("%s/%s  compare %s...%s", m.currentURL.Owner, m.currentURL.Repo, m.currentURL.CompareBase, m.currentURL.CompareHead)
		case gh.TargetPullRequest:
			info = fmt.Sprintf("%s/%s  pull #%d @ %s", m.currentURL.Owner, m.currentURL.Repo, m.currentURL.PullNumber, m.currentURL.Branch)
		default:
			info = m.currentURL.Owner + "/" + m.currentURL.Repo + " @ " + m.currentURL.Branch + "  /"
			if m.currentURL.Path != "" {
				info = m.currentURL.Owner + "/" + m.currentURL.Repo + " @ " + m.currentURL.Branch + "  /" + m.currentURL.Path
			}
		}
	}

	selected := ""
	if count := len(m.selectedPath); count > 0 {
		selected = baseTextStyle.Render("  ") + successText.Render(fmt.Sprintf("[%d selected]", count))
	}

	favorite := ""
	if m.isFavoriteRepo() {
		favorite = baseTextStyle.Render("  ") + successText.Render("[favorite]")
	}
	status := ""
	if auth := m.authStatusLabel(); auth != "" {
		status = baseTextStyle.Render("  ") + mutedTextStyle.Render("["+auth+"]")
	}

	title := accentTextStyle.Render(" Repository ")
	content := headerTextStyle.Render(info) + favorite + selected + status

	return titledPanelWithColor(title, content, panelWidth, colorAccent)
}

func (m *model) renderFileList(panelWidth int) string {
	items := m.viewItems()

	title := panelTitle(fmt.Sprintf(" Files (%d) ", len(items)))

	if len(items) == 0 {
		return titledPanelWithColor(title, mutedTextStyle.Render("No files to display."), panelWidth, colorBorder)
	}

	nameWidth := max(20, min(panelWidth-36, 60))
	lines := []string{
		headerTextStyle.Render(fmt.Sprintf("     %-*s  %-7s %10s", nameWidth, "Name", "Type", "Size")),
	}

	start := min(m.scrollOffset, max(len(items)-1, 0))
	end := min(start+m.visibleListHeight(), len(items))
	for index := start; index < end; index++ {
		item := items[index]

		// Selection checkbox
		checkbox := "[ ] "
		if item.Selected && index == m.cursor {
			checkbox = "[◉] "
		} else if item.Selected {
			checkbox = "[●] "
		} else if index == m.cursor {
			checkbox = "[◎] "
		}

		typeLabel := itemTypeLabel(item)
		line := fmt.Sprintf(
			"%s%-*s  %-7s %10s",
			checkbox,
			nameWidth,
			truncate(itemLabel(item, m.asciiMode, m.searching), nameWidth),
			typeLabel,
			sizeLabel(item, m.folderSizes),
		)

		switch {
		case index == m.cursor:
			lines = append(lines, selectedRow.Render(line))
		case item.IsDir():
			lines = append(lines, folderText.Render(line))
		default:
			lines = append(lines, line)
		}
	}

	return titledPanelWithColor(title, strings.Join(lines, "\n"), panelWidth, colorBorder)
}

func (m *model) renderDownloadProgress() string {
	completed := m.downloadProgress.Completed()
	total := max(m.downloadProgress.Total, 1)
	percentage := completed * 100 / total
	barWidth := max(12, min(m.width-28, 48))
	filled := barWidth * percentage / 100
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	fileName := nonEmpty(m.downloadProgress.CurrentFile(), "Starting...")

	body := []string{
		successText.Render(fmt.Sprintf("[%s] %3d%%  %d/%d", bar, percentage, completed, total)),
		mutedTextStyle.Render(truncate(fileName, 72)),
	}

	return panelWithColor(strings.Join(body, "\n"), max(44, min(fullPanelWidth(m.width), 72)), colorBorder)
}

func itemLabel(item gh.RepoItem, asciiMode, searching bool) string {
	name := item.Name
	if searching {
		name = item.Path
	}

	if asciiMode {
		switch {
		case item.IsDir():
			return "[D] " + name
		case item.IsLFS():
			return "[L] " + name
		default:
			return "[F] " + name
		}
	}

	switch {
	case item.IsDir():
		return "◆ " + name
	case item.IsLFS():
		return "◉ " + name
	default:
		return "• " + name
	}
}

func itemTypeLabel(item gh.RepoItem) string {
	switch {
	case item.IsDir():
		return "DIR"
	case item.IsLFS():
		return "LFS"
	default:
		return fileExtLabel(item.Name)
	}
}

func sizeLabel(item gh.RepoItem, folderSizes map[string]uint64) string {
	switch {
	case item.IsDir():
		return humanSize(folderSizes[item.Path])
	default:
		return humanSize(item.ActualSize())
	}
}

func humanSize(size uint64) string {
	if size == 0 {
		return "-"
	}

	units := []string{"B", "KB", "MB", "GB"}
	value := float64(size)
	index := 0
	for value >= 1024 && index < len(units)-1 {
		value /= 1024
		index++
	}

	switch {
	case value >= 10 || index == 0:
		return fmt.Sprintf("%.0f %s", value, units[index])
	default:
		return fmt.Sprintf("%.1f %s", value, units[index])
	}
}

func truncate(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}
	return lipgloss.NewStyle().MaxWidth(width-1).Render(value) + "…"
}

func panelTitle(text string) string {
	return successText.Render(text)
}
