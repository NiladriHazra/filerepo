package ui

import (
	"strings"
	"time"

	gh "github.com/NiladriHazra/filerepo/internal/github"
)

const (
	listHeaderRows    = 3
	helpRows          = 2
	progressRows      = 3
	searchRows        = 3
	maxPreviewBytes   = 256 * 1024
	maxPreviewChars   = 40_000
	tickInterval      = 120 * time.Millisecond
	defaultWindowW    = 108
	defaultWindowH    = 32
	defaultToastTTL   = 4 * time.Second
	defaultPageStep   = 10
	defaultListHeight = 10
)

func newModel(initialURL, token, downloadPath string, cwd, noFolder bool) *model {
	input := initialURL
	if input == "" {
		input = gh.GetLocalGitRemote()
	}

	return &model{
		width:                  defaultWindowW,
		height:                 defaultWindowH,
		mode:                   modeInput,
		urlInput:               input,
		urlCursor:              len(input),
		client:                 gh.NewClient(token),
		sessionToken:           token,
		configuredDownloadPath: downloadPath,
		cwd:                    cwd,
		noFolder:               noFolder,
		folderSizes:            map[string]uint64{},
		selectedPath:           map[string]struct{}{},
	}
}

func (m *model) visibleListHeight() int {
	height := m.height - listHeaderRows - helpRows - 2
	if m.downloading {
		height -= progressRows
	}
	if m.searching {
		height -= searchRows
	}
	if height < defaultListHeight {
		return defaultListHeight
	}
	return height
}

func (m *model) resetBrowserPosition(cursor int) {
	m.cursor = clampCursor(cursor, len(m.viewItems()))
	m.scrollOffset = 0
	m.adjustScroll()
}

func (m *model) adjustScroll() {
	height := m.visibleListHeight()
	switch {
	case m.cursor < m.scrollOffset:
		m.scrollOffset = m.cursor
	case m.cursor >= m.scrollOffset+height:
		m.scrollOffset = m.cursor - height + 1
	}
}

func (m *model) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
	m.adjustScroll()
}

func (m *model) moveDown() {
	last := len(m.viewItems()) - 1
	if last >= 0 && m.cursor < last {
		m.cursor++
	}
	m.adjustScroll()
}

func (m *model) moveTop() {
	m.cursor = 0
	m.adjustScroll()
}

func (m *model) moveBottom() {
	items := m.viewItems()
	if len(items) == 0 {
		return
	}
	m.cursor = len(items) - 1
	m.adjustScroll()
}

func (m *model) moveBy(delta int) {
	switch {
	case delta < 0:
		m.cursor = max(m.cursor+delta, 0)
	default:
		m.cursor = min(m.cursor+delta, max(len(m.viewItems())-1, 0))
	}
	m.adjustScroll()
}

func (m *model) clearSearch() {
	m.searching = false
	m.searchQuery = ""
	m.resetBrowserPosition(0)
}

func (m *model) showToast(message string, kind toastKind) {
	m.toast = &toast{
		message:   message,
		kind:      kind,
		expiresAt: time.Now().Add(defaultToastTTL),
	}
}

func (m *model) viewItems() []gh.RepoItem {
	var items []gh.RepoItem
	if m.searching {
		source := m.items
		if m.hasFullTree {
			source = m.fullTree
		}

		query := strings.ToLower(m.searchQuery)
		items = filterItems(source, func(item gh.RepoItem) bool {
			return strings.Contains(strings.ToLower(item.Path), query)
		})
		sortItemsByPath(items)
	} else {
		items = cloneItems(m.items)
	}

	for index := range items {
		_, ok := m.selectedPath[items[index].Path]
		items[index].Selected = ok
	}

	return items
}

func (m *model) toggleSelectionAtCursor() {
	items := m.viewItems()
	if m.cursor >= len(items) {
		return
	}

	item := items[m.cursor]
	if _, ok := m.selectedPath[item.Path]; ok {
		delete(m.selectedPath, item.Path)
		return
	}

	m.selectedPath[item.Path] = struct{}{}
}

func (m *model) selectVisible(selected bool) {
	for _, item := range m.viewItems() {
		switch selected {
		case true:
			m.selectedPath[item.Path] = struct{}{}
		default:
			delete(m.selectedPath, item.Path)
		}
	}
}

func (m *model) selectedItems() []gh.RepoItem {
	source := m.items
	if m.hasFullTree {
		source = m.fullTree
	}

	return filterItems(source, func(item gh.RepoItem) bool {
		_, ok := m.selectedPath[item.Path]
		return ok
	})
}

func (m *model) selectedOrFocusedItems() []gh.RepoItem {
	selected := m.selectedItems()
	if len(selected) > 0 {
		return selected
	}

	items := m.viewItems()
	if m.cursor >= len(items) {
		return nil
	}

	item := items[m.cursor]
	item.Selected = true
	return []gh.RepoItem{item}
}
