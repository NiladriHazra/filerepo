package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	gh "github.com/NiladriHazra/filerepo/internal/github"
)

type searchOptions struct {
	text       string
	ext        string
	typeFilter string
}

func cloneItems(items []gh.RepoItem) []gh.RepoItem {
	return append([]gh.RepoItem(nil), items...)
}

func filterItems(items []gh.RepoItem, keep func(gh.RepoItem) bool) []gh.RepoItem {
	filtered := make([]gh.RepoItem, 0, len(items))
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortVisibleItems(items []gh.RepoItem) {
	slices.SortFunc(items, func(a, b gh.RepoItem) int {
		switch {
		case a.IsDir() && !b.IsDir():
			return -1
		case !a.IsDir() && b.IsDir():
			return 1
		default:
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		}
	})
}

func sortItemsByPath(items []gh.RepoItem) {
	slices.SortFunc(items, func(a, b gh.RepoItem) int {
		return strings.Compare(a.Path, b.Path)
	})
}

func clampCursor(cursor, count int) int {
	if count == 0 {
		return 0
	}
	return min(cursor, count-1)
}

func repoItemsForPath(items []gh.RepoItem, currentPath string) []gh.RepoItem {
	switch currentPath {
	case "":
		rootItems := filterItems(items, func(item gh.RepoItem) bool {
			return !strings.Contains(item.Path, "/")
		})
		sortVisibleItems(rootItems)
		return rootItems
	default:
		prefix := currentPath + "/"
		children := filterItems(items, func(item gh.RepoItem) bool {
			if !strings.HasPrefix(item.Path, prefix) {
				return false
			}
			return !strings.Contains(item.Path[len(prefix):], "/")
		})
		sortVisibleItems(children)
		return children
	}
}

func calculateFolderSizes(items []gh.RepoItem) map[string]uint64 {
	sizes := map[string]uint64{}
	for _, item := range items {
		if !item.IsFile() {
			continue
		}

		parts := strings.Split(item.Path, "/")
		for index := 1; index < len(parts); index++ {
			parent := strings.Join(parts[:index], "/")
			sizes[parent] += item.ActualSize()
		}
	}
	return sizes
}

func resolveRequestedView(items []gh.RepoItem, requestedPath string) (string, string) {
	if requestedPath == "" {
		return "", ""
	}

	for _, item := range items {
		if item.Path == requestedPath && item.IsFile() {
			return parentRepoPath(requestedPath), requestedPath
		}
	}

	return requestedPath, ""
}

func parentRepoPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], "/")
}

func findCursorByPath(items []gh.RepoItem, target string) int {
	for index, item := range items {
		if item.Path == target {
			return index
		}
	}
	return 0
}

func isExactFileMatch(items []gh.RepoItem, requestedPath string) bool {
	return len(items) == 1 && items[0].IsFile() && items[0].Path == requestedPath
}

func downloadableItem(item gh.RepoItem) gh.RepoItem {
	item.Name = item.Path
	item.Selected = true
	return item
}

func dedupeItems(items []gh.RepoItem) []gh.RepoItem {
	seen := map[string]struct{}{}
	unique := make([]gh.RepoItem, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.Path]; ok {
			continue
		}
		seen[item.Path] = struct{}{}
		unique = append(unique, item)
	}
	sortItemsByPath(unique)
	return unique
}

func mapTreeToItems(tree gh.GitTreeResponse, owner, repo, branch string) []gh.RepoItem {
	items := make([]gh.RepoItem, 0, len(tree.Tree))
	for _, entry := range tree.Tree {
		itemType := "file"
		if entry.EntryType == "tree" {
			itemType = "dir"
		}

		name := entry.Path
		if slash := strings.LastIndex(entry.Path, "/"); slash >= 0 {
			name = entry.Path[slash+1:]
		}

		downloadURL := ""
		if itemType == "file" {
			downloadURL = fmt.Sprintf(
				"https://raw.githubusercontent.com/%s/%s/%s/%s",
				owner,
				repo,
				branch,
				entry.Path,
			)
		}

		items = append(items, gh.RepoItem{
			Name:        name,
			ItemType:    itemType,
			Path:        entry.Path,
			URL:         fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, entry.Path, branch),
			DownloadURL: downloadURL,
			Size:        entry.Size,
		})
	}
	return items
}

func mapCompareFilesToItems(files []gh.CompareFile) []gh.RepoItem {
	items := make([]gh.RepoItem, 0, len(files))
	for _, file := range files {
		if stringsTrimmed(file.RawURL) == "" {
			continue
		}

		name := file.Filename
		if slash := strings.LastIndex(file.Filename, "/"); slash >= 0 {
			name = file.Filename[slash+1:]
		}

		items = append(items, gh.RepoItem{
			Name:        name,
			ItemType:    "file",
			Path:        file.Filename,
			DownloadURL: file.RawURL,
			HTMLURL:     file.BlobURL,
			Status:      file.Status,
			TargetKind:  "compare",
		})
	}

	sortItemsByPath(items)
	return items
}

func collectDownloadItems(ctx context.Context, client *gh.Client, selectedItems, fullTree []gh.RepoItem, hasFullTree bool) ([]gh.RepoItem, error) {
	var items []gh.RepoItem
	for _, item := range selectedItems {
		switch {
		case item.IsFile():
			items = append(items, downloadableItem(item))
		case hasFullTree:
			prefix := item.Path + "/"
			for _, treeItem := range fullTree {
				if treeItem.IsFile() && strings.HasPrefix(treeItem.Path, prefix) {
					items = append(items, downloadableItem(treeItem))
				}
			}
		default:
			files, err := collectDirectoryFiles(ctx, client, item)
			if err != nil {
				return nil, err
			}
			items = append(items, files...)
		}
	}
	return dedupeItems(items), nil
}

func collectDirectoryFiles(ctx context.Context, client *gh.Client, root gh.RepoItem) ([]gh.RepoItem, error) {
	pending := []gh.RepoItem{root}
	files := make([]gh.RepoItem, 0, 16)

	for len(pending) > 0 {
		last := len(pending) - 1
		current := pending[last]
		pending = pending[:last]

		items, err := client.FetchContents(ctx, current.URL)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			if item.IsFile() {
				files = append(files, downloadableItem(item))
				continue
			}
			pending = append(pending, item)
		}
	}

	return files, nil
}

func defaultDownloadDir() (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	return dir, nil
}

func stringsTrimmed(value string) string {
	return strings.TrimSpace(value)
}

func parseSearchQuery(query string) searchOptions {
	options := searchOptions{}
	var terms []string

	for _, token := range strings.Fields(strings.ToLower(query)) {
		switch {
		case strings.HasPrefix(token, "ext:"):
			options.ext = strings.TrimPrefix(token, "ext:")
			options.ext = strings.TrimPrefix(options.ext, ".")
		case strings.HasPrefix(token, "type:"):
			options.typeFilter = strings.TrimPrefix(token, "type:")
		case strings.HasPrefix(token, "kind:"):
			options.typeFilter = strings.TrimPrefix(token, "kind:")
		default:
			terms = append(terms, token)
		}
	}

	options.text = strings.Join(terms, " ")
	return options
}

func matchesSearch(item gh.RepoItem, options searchOptions) bool {
	switch options.typeFilter {
	case "", "any":
	case "dir", "folder":
		if !item.IsDir() {
			return false
		}
	case "file":
		if !item.IsFile() || item.IsLFS() {
			return false
		}
	case "lfs":
		if !item.IsLFS() {
			return false
		}
	default:
		return false
	}

	if options.ext != "" && fileExtLower(item.Name) != options.ext {
		return false
	}

	if options.text == "" {
		return true
	}

	path := strings.ToLower(item.Path)
	if strings.Contains(path, options.text) {
		return true
	}
	return fuzzyMatch(path, options.text)
}

func fuzzyMatch(value, query string) bool {
	valueRunes := []rune(strings.ToLower(value))
	queryRunes := []rune(strings.ToLower(strings.ReplaceAll(query, " ", "")))
	if len(queryRunes) == 0 {
		return true
	}

	index := 0
	for _, current := range valueRunes {
		if current != unicode.ToLower(queryRunes[index]) {
			continue
		}
		index++
		if index == len(queryRunes) {
			return true
		}
	}

	return false
}

// fileExtLabel returns a short uppercase type tag based on file extension.
func fileExtLabel(name string) string {
	dot := strings.LastIndex(name, ".")
	if dot < 0 || dot == len(name)-1 {
		return "FILE"
	}
	ext := strings.ToUpper(name[dot+1:])
	switch ext {
	case "RS", "PY", "JS", "TS", "GO", "RB", "C", "CPP", "H", "JAVA",
		"MD", "TXT", "JSON", "YAML", "YML", "TOML",
		"CSS", "HTML", "SCSS", "LESS",
		"SH", "BAT", "PS1",
		"MOD", "SUM", "LOCK",
		"PNG", "JPG", "JPEG", "GIF", "SVG", "ICO",
		"ZIP", "TAR", "GZ",
		"XML", "CSV", "SQL",
		"DOCKERFILE", "MAKEFILE":
		return ext
	}
	return "FILE"
}

func fileExtLower(name string) string {
	dot := strings.LastIndex(name, ".")
	if dot < 0 || dot == len(name)-1 {
		return ""
	}
	return strings.ToLower(name[dot+1:])
}
