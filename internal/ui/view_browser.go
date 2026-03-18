package ui

import (
	"fmt"
	"strings"

	gh "github.com/NiladriHazra/filerepo/internal/github"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderBrowser() string {
	sections := []string{
		m.renderRepositoryHeader(),
		m.renderFileList(),
	}

	if m.downloading && m.downloadProgress != nil {
		sections = append(sections, m.renderDownloadProgress())
	}
	if m.searching {
		sections = append(sections, accentPanelStyle.Render("Search\n/"+m.searchQuery+"_"))
	}

	sections = append(sections, mutedTextStyle.Render("Enter/open  h/back  space/select  d/download  /search  i/icons  q/quit"))
	return strings.Join(sections, "\n\n")
}

func (m *model) renderRepositoryHeader() string {
	path := "Repository"
	if m.currentURL != nil {
		path = m.currentURL.Repo
	}
	if m.currentURL != nil && m.currentURL.Path != "" {
		path = m.currentURL.Repo + " / " + m.currentURL.Path
	}

	selected := ""
	if count := len(m.selectedPath); count > 0 {
		selected = "  [" + fmt.Sprintf("%d selected", count) + "]"
	}

	return accentPanelStyle.Render(headerTextStyle.Render(path) + successText.Render(selected))
}

func (m *model) renderFileList() string {
	items := m.viewItems()
	if len(items) == 0 {
		return panelStyle.Render("Files\n\n" + mutedTextStyle.Render("No files to display."))
	}

	nameWidth := max(20, min(m.width-36, 60))
	lines := []string{
		headerTextStyle.Render(fmt.Sprintf("%-4s %-*s %-7s %10s", "", nameWidth, "Name", "Type", "Size")),
	}

	start := min(m.scrollOffset, max(len(items)-1, 0))
	end := min(start+m.visibleListHeight(), len(items))
	for index := start; index < end; index++ {
		item := items[index]
		prefix := "  "
		if item.Selected {
			prefix = "■ "
		}
		if index == m.cursor {
			prefix = "▶ "
		}

		line := fmt.Sprintf(
			"%-4s %-*s %-7s %10s",
			prefix,
			nameWidth,
			truncate(itemLabel(item, m.asciiMode, m.searching), nameWidth),
			itemTypeLabel(item),
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

	return panelStyle.Width(max(60, min(m.width-6, 110))).Render(strings.Join(lines, "\n"))
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

	return panelStyle.Render(strings.Join(body, "\n"))
}

func itemLabel(item gh.RepoItem, asciiMode, searching bool) string {
	name := item.Name
	if searching {
		name = item.Path
	}

	if asciiMode {
		switch {
		case item.IsDir():
			return "[DIR] " + name
		case item.IsLFS():
			return "[LFS] " + name
		default:
			return "[FIL] " + name
		}
	}

	switch {
	case item.IsDir():
		return "󰉋 " + name
	case item.IsLFS():
		return "󰈈 " + name
	default:
		return "󰈔 " + name
	}
}

func itemTypeLabel(item gh.RepoItem) string {
	switch {
	case item.IsDir():
		return "dir"
	case item.IsLFS():
		return "lfs"
	default:
		return "file"
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
		return fmt.Sprintf("%.0f%s", value, units[index])
	default:
		return fmt.Sprintf("%.1f%s", value, units[index])
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
