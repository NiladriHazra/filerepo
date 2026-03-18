package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleBrowseMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.savePrompt != nil:
		return m.handleSavePrompt(msg)
	case m.preview != nil:
		return m.handlePreview(msg)
	case m.searching:
		return m.handleSearch(msg)
	default:
		return m.handleBrowserKeys(msg)
	}
}

func (m *model) handlePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.preview = nil
	case "up", "k":
		if m.preview.scroll > 0 {
			m.preview.scroll--
		}
	case "down", "j":
		m.preview.scroll++
	case "pgup":
		m.preview.scroll = max(m.preview.scroll-defaultPageStep, 0)
	case "pgdown":
		m.preview.scroll += defaultPageStep
	case "home", "g":
		m.preview.scroll = 0
	}

	return m, nil
}

func (m *model) handleSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.clearSearch()
	case "enter":
		m.searching = false
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.resetBrowserPosition(0)
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
			m.resetBrowserPosition(0)
		}
	}

	return m, nil
}

func (m *model) handleBrowserKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.mode = modeInput
		m.selectedPath = map[string]struct{}{}
		return m, nil
	case "i":
		m.asciiMode = !m.asciiMode
		switch m.asciiMode {
		case true:
			m.showToast("ASCII mode.", toastInfo)
		default:
			m.showToast("Icon mode.", toastInfo)
		}
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "home", "g":
		m.moveTop()
	case "end", "G":
		m.moveBottom()
	case "pgup":
		m.moveBy(-defaultPageStep)
	case "pgdown":
		m.moveBy(defaultPageStep)
	case " ":
		m.toggleSelectionAtCursor()
	case "a":
		m.selectVisible(true)
	case "u":
		m.selectVisible(false)
	case "/":
		m.searching = true
		m.searchQuery = ""
		m.resetBrowserPosition(0)
	case "backspace", "left", "h":
		return m.navigateBack()
	case "enter", "right", "l":
		return m.openCurrentItem()
	case "d":
		items := m.selectedOrFocusedItems()
		if len(items) == 0 {
			m.showToast("Nothing to download here.", toastWarning)
			return m, nil
		}

		defaultPath, err := defaultDownloadDir()
		if err != nil {
			m.showToast(err.Error(), toastError)
			return m, nil
		}

		m.savePrompt = &savePrompt{
			input:     defaultPath,
			cursor:    len(defaultPath),
			itemCount: len(items),
		}
	}

	return m, nil
}

func (m *model) navigateBack() (tea.Model, tea.Cmd) {
	if len(m.navigation) == 0 {
		m.mode = modeInput
		return m, nil
	}

	last := m.navigation[len(m.navigation)-1]
	m.navigation = m.navigation[:len(m.navigation)-1]

	if m.hasFullTree {
		items := repoItemsForPath(m.fullTree, last.url.Path)
		m.currentURL = &last.url
		m.items = items
		m.resetBrowserPosition(last.cursor)
		return m, nil
	}

	m.mode = modeLoading
	m.statusMessage = "Loading directory..."
	m.urlInput = fmt.Sprintf("https://github.com/%s/%s/tree/%s/%s", last.url.Owner, last.url.Repo, last.url.Branch, last.url.Path)
	return m, loadRepoCmd(m.urlInput, m.sessionToken)
}

func (m *model) openCurrentItem() (tea.Model, tea.Cmd) {
	items := m.viewItems()
	if m.cursor >= len(items) {
		return m, nil
	}

	item := items[m.cursor]
	if item.IsFile() {
		m.preview = &previewState{
			path:   item.Path,
			status: previewLoading,
		}
		return m, fetchPreviewCmd(item, m.sessionToken)
	}

	current := *m.currentURL
	m.navigation = append(m.navigation, navState{url: current, cursor: m.cursor})
	nextURL := current
	nextURL.Path = item.Path

	if m.hasFullTree {
		m.currentURL = &nextURL
		m.items = repoItemsForPath(m.fullTree, item.Path)
		m.resetBrowserPosition(0)
		return m, nil
	}

	target := fmt.Sprintf("https://github.com/%s/%s/tree/%s/%s", nextURL.Owner, nextURL.Repo, nextURL.Branch, nextURL.Path)
	m.urlInput = target
	m.mode = modeLoading
	m.statusMessage = "Loading directory..."
	return m, loadRepoCmd(target, m.sessionToken)
}
