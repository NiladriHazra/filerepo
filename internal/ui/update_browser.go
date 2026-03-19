package ui

import (
	"fmt"

	"github.com/NiladriHazra/filerepo/internal/download"
	gh "github.com/NiladriHazra/filerepo/internal/github"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleBrowseMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.savePrompt != nil:
		return m.handleSavePrompt(msg)
	case m.preview != nil:
		return m.handlePreview(msg)
	case m.refPicker != nil:
		return m.handleRefPicker(msg)
	case m.releasePicker != nil:
		return m.handleReleasePicker(msg)
	case m.infoOverlay != nil:
		return m.handleInfoOverlay(msg)
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
	case "w":
		m.preview.wrap = !m.preview.wrap
	case "n":
		m.preview.showNumbers = !m.preview.showNumbers
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

func (m *model) handleRefPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.refPicker = nil
	case "enter":
		refs := m.filteredRefs()
		if len(refs) == 0 || m.currentURL == nil {
			return m, nil
		}

		selected := refs[min(m.refPicker.cursor, len(refs)-1)]
		m.refPicker = nil
		m.navigation = nil
		m.selectedPath = map[string]struct{}{}
		target := *m.currentURL
		target.Branch = selected.Name
		target.Path = ""
		m.urlInput = target.WebURL()
		m.mode = modeLoading
		m.statusMessage = fmt.Sprintf("Switching to %s...", selected.Name)
		return m, loadRepoCmd(m.urlInput, m.sessionToken, m.configState.Cache.Enabled, m.configState.CacheTTL())
	case "up", "k":
		if m.refPicker.cursor > 0 {
			m.refPicker.cursor--
		}
	case "down", "j":
		if refs := m.filteredRefs(); m.refPicker.cursor < len(refs)-1 {
			m.refPicker.cursor++
		}
	case "backspace":
		if len(m.refPicker.query) > 0 {
			m.refPicker.query = m.refPicker.query[:len(m.refPicker.query)-1]
			m.refPicker.cursor = 0
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.refPicker.query += string(msg.Runes)
			m.refPicker.cursor = 0
		}
	}

	return m, nil
}

func (m *model) handleInfoOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.infoOverlay = nil
	case "tab", "right", "l":
		m.infoOverlay.tab = (m.infoOverlay.tab + 1) % 3
		m.infoOverlay.scroll = 0
	case "shift+tab", "left", "h":
		m.infoOverlay.tab = (m.infoOverlay.tab + 2) % 3
		m.infoOverlay.scroll = 0
	case "up", "k":
		if m.infoOverlay.scroll > 0 {
			m.infoOverlay.scroll--
		}
	case "down", "j":
		m.infoOverlay.scroll++
	case "home", "g":
		m.infoOverlay.scroll = 0
	case "pgup":
		m.infoOverlay.scroll = max(m.infoOverlay.scroll-defaultPageStep, 0)
	case "pgdown":
		m.infoOverlay.scroll += defaultPageStep
	}

	return m, nil
}

func (m *model) handleReleasePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	assets := m.releaseItems()
	switch msg.String() {
	case "esc", "q":
		m.releasePicker = nil
	case "up", "k":
		if m.releasePicker.cursor > 0 {
			m.releasePicker.cursor--
		}
	case "down", "j":
		if m.releasePicker.cursor < len(assets)-1 {
			m.releasePicker.cursor++
		}
	case "enter", "d":
		if len(assets) == 0 {
			return m, nil
		}
		defaultPath, err := defaultDownloadDir()
		if err != nil {
			m.showToast(err.Error(), toastError)
			return m, nil
		}
		asset := assets[m.releasePicker.cursor]
		m.releasePicker = nil
		m.savePrompt = &savePrompt{
			input:     defaultPath,
			cursor:    len(defaultPath),
			itemCount: 1,
			conflict:  download.ConflictSkip,
			output:    download.OutputFiles,
			items:     []gh.RepoItem{asset},
		}
	case "y":
		if len(assets) == 0 {
			return m, nil
		}
		if err := copyText(assets[m.releasePicker.cursor].DownloadURL); err != nil {
			m.showToast(err.Error(), toastError)
		} else {
			m.showToast("Copied release asset URL.", toastSuccess)
		}
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
	case "b":
		m.openRefPicker("branch")
	case "t":
		m.openRefPicker("tag")
	case "m":
		m.infoOverlay = &infoOverlayState{}
	case "R":
		if len(m.releaseItems()) == 0 {
			m.showToast("No release assets available.", toastWarning)
		} else {
			m.releasePicker = &releasePickerState{}
		}
	case "f":
		switch m.toggleFavorite(m.currentRepoURL) {
		case true:
			m.showToast("Repository added to favorites.", toastSuccess)
		default:
			m.showToast("Repository removed from favorites.", toastInfo)
		}
	case "y":
		if err := copyText(m.currentItemWebURL()); err != nil {
			m.showToast(err.Error(), toastError)
		} else {
			m.showToast("Copied GitHub URL.", toastSuccess)
		}
	case "Y":
		if err := copyText(m.currentItemRawURL()); err != nil {
			m.showToast(err.Error(), toastError)
		} else {
			m.showToast("Copied raw URL.", toastSuccess)
		}
	case "P":
		if err := copyText(m.plannedOutputPath()); err != nil {
			m.showToast(err.Error(), toastError)
		} else {
			m.showToast("Copied planned output path.", toastSuccess)
		}
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
			conflict:  download.ConflictSkip,
			output:    download.OutputFiles,
			items:     items,
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
	return m, loadRepoCmd(m.urlInput, m.sessionToken, m.configState.Cache.Enabled, m.configState.CacheTTL())
}

func (m *model) openCurrentItem() (tea.Model, tea.Cmd) {
	items := m.viewItems()
	if m.cursor >= len(items) {
		return m, nil
	}

	item := items[m.cursor]
	if item.IsFile() {
		m.preview = &previewState{
			path:        item.Path,
			status:      previewLoading,
			showNumbers: true,
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
	return m, loadRepoCmd(target, m.sessionToken, m.configState.Cache.Enabled, m.configState.CacheTTL())
}
