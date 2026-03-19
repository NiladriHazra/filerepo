package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	gh "github.com/NiladriHazra/filerepo/internal/github"
)

func (m *model) filteredRefs() []gh.RepoRef {
	if m.refPicker == nil {
		return nil
	}

	query := strings.ToLower(strings.TrimSpace(m.refPicker.query))
	refs := filterItemsByRef(m.repoRefs, func(ref gh.RepoRef) bool {
		if m.refPicker.kind != "" && ref.Kind != m.refPicker.kind {
			return false
		}
		if query == "" {
			return true
		}
		return strings.Contains(strings.ToLower(ref.Name), query) || fuzzyMatch(strings.ToLower(ref.Name), query)
	})
	return refs
}

func filterItemsByRef(refs []gh.RepoRef, keep func(gh.RepoRef) bool) []gh.RepoRef {
	filtered := make([]gh.RepoRef, 0, len(refs))
	for _, ref := range refs {
		if keep(ref) {
			filtered = append(filtered, ref)
		}
	}
	return filtered
}

func (m *model) focusedItem() (gh.RepoItem, bool) {
	items := m.viewItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return gh.RepoItem{}, false
	}
	return items[m.cursor], true
}

func (m *model) openRefPicker(kind string) {
	if len(m.repoRefs) == 0 {
		m.showToast("No refs loaded for this repository.", toastWarning)
		return
	}

	m.refPicker = &refPickerState{kind: kind}
}

func (m *model) currentItemWebURL() string {
	item, ok := m.focusedItem()
	if !ok || m.currentURL == nil {
		return m.currentRepoURL
	}

	if item.HTMLURL != "" {
		return item.HTMLURL
	}

	base := m.currentURL.RepoURL()
	if item.Path == "" {
		return base
	}
	return fmt.Sprintf("%s/blob/%s/%s", base, m.currentURL.Branch, item.Path)
}

func (m *model) currentItemRawURL() string {
	item, ok := m.focusedItem()
	if !ok {
		return m.currentRepoURL
	}

	if item.IsDir() {
		return m.currentRepoURL
	}

	return item.ActualDownloadURL()
}

func (m *model) plannedOutputPath() string {
	item, ok := m.focusedItem()
	if !ok || m.currentURL == nil {
		return ""
	}

	baseDir := m.configuredDownloadPath
	if baseDir == "" {
		dir, err := defaultDownloadDir()
		if err == nil {
			baseDir = dir
		}
	}
	if baseDir == "" {
		return ""
	}
	if !m.noFolder {
		baseDir = filepath.Join(baseDir, m.currentURL.Repo)
	}
	return filepath.Join(baseDir, filepath.FromSlash(item.Path))
}

func (m *model) authStatusLabel() string {
	if m.rateLimit.Limit == 0 {
		if m.sessionToken != "" {
			return "auth"
		}
		return "public"
	}

	label := "public"
	if m.rateLimit.Authenticated {
		label = "auth"
		if m.activeProfile != "" {
			label = "auth:" + m.activeProfile
		}
	}

	return fmt.Sprintf("%s %d/%d", label, m.rateLimit.Remaining, m.rateLimit.Limit)
}

func (m *model) isFavoriteRepo() bool {
	return m.configState.IsFavorite(m.currentRepoURL)
}

func (m *model) releaseItems() []gh.RepoItem {
	items := make([]gh.RepoItem, 0)
	for _, release := range m.repoReleases {
		for _, asset := range release.Assets {
			size := asset.Size
			items = append(items, gh.RepoItem{
				Name:        asset.Name,
				ItemType:    "file",
				Path:        filepath.ToSlash(filepath.Join("release-assets", release.TagName, asset.Name)),
				DownloadURL: asset.DownloadURL,
				HTMLURL:     asset.DownloadURL,
				Size:        &size,
				Status:      release.TagName,
				TargetKind:  "release",
			})
		}
	}
	sortItemsByPath(items)
	return items
}
