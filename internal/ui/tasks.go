package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/NiladriHazra/filerepo/internal/download"
	gh "github.com/NiladriHazra/filerepo/internal/github"
	tea "github.com/charmbracelet/bubbletea"
)

type loadResult struct {
	currentURL   gh.URL
	items        []gh.RepoItem
	fullTree     []gh.RepoItem
	hasFullTree  bool
	folderSizes  map[string]uint64
	cursor       int
	warning      string
	sessionToken string
}

type downloadRequest struct {
	currentURL   gh.URL
	selected     []gh.RepoItem
	fullTree     []gh.RepoItem
	hasFullTree  bool
	token        string
	configPath   string
	overridePath string
	cwd          bool
	noFolder     bool
	progress     *download.Progress
}

func loadRepoCmd(rawURL, token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		parsed, err := gh.ParseURL(stringsTrimmed(rawURL))
		if err != nil {
			return repoLoadFailedMsg{err: fmt.Errorf("invalid URL: %w", err)}
		}

		result, err := loadRepository(ctx, parsed, token)
		if err != nil {
			return repoLoadFailedMsg{err: err}
		}

		return repoLoadedMsg{
			currentURL:   result.currentURL,
			items:        result.items,
			fullTree:     result.fullTree,
			hasFullTree:  result.hasFullTree,
			folderSizes:  result.folderSizes,
			cursor:       result.cursor,
			warning:      result.warning,
			sessionToken: result.sessionToken,
		}
	}
}

func loadRepository(ctx context.Context, target gh.URL, token string) (loadResult, error) {
	client := gh.NewClient(token)
	sessionToken := token
	warning := ""

	tree, err := client.FetchRecursiveTree(ctx, target.Owner, target.Repo, target.Branch)
	var apiErr *gh.APIError
	if errors.As(err, &apiErr) && apiErr.Kind == gh.ErrorInvalidToken && token != "" {
		client = gh.NewClient("")
		sessionToken = ""
		warning = "Invalid token. Falling back to public API."
		tree, err = client.FetchRecursiveTree(ctx, target.Owner, target.Repo, target.Branch)
	}

	if errors.As(err, &apiErr) && apiErr.Kind == gh.ErrorNotFound && target.Branch == "main" {
		target.Branch = "master"
		tree, err = client.FetchRecursiveTree(ctx, target.Owner, target.Repo, target.Branch)
	}

	switch {
	case err == nil && !tree.Truncated:
		allItems := mapTreeToItems(tree, target.Owner, target.Repo, target.Branch)
		browsePath, cursorPath := resolveRequestedView(allItems, target.Path)
		currentItems := repoItemsForPath(allItems, browsePath)
		client.ResolveLFSFiles(ctx, currentItems, target.Owner, target.Repo, target.Branch)

		cursor := 0
		if cursorPath != "" {
			cursor = findCursorByPath(currentItems, cursorPath)
		}

		target.Path = browsePath
		return loadResult{
			currentURL:   target,
			items:        currentItems,
			fullTree:     allItems,
			hasFullTree:  true,
			folderSizes:  calculateFolderSizes(allItems),
			cursor:       cursor,
			warning:      warning,
			sessionToken: sessionToken,
		}, nil
	default:
		return loadFolderView(ctx, client, target, warning, sessionToken)
	}
}

func loadFolderView(ctx context.Context, client *gh.Client, target gh.URL, warning, sessionToken string) (loadResult, error) {
	requestedPath := target.Path
	selectedFilePath := requestedPath

	items, err := client.FetchContents(ctx, target.APIURL())
	if err == nil && isExactFileMatch(items, requestedPath) {
		target.Path = parentRepoPath(requestedPath)
		items, err = client.FetchContents(ctx, target.APIURL())
	}
	if err != nil {
		return loadResult{}, err
	}

	sortVisibleItems(items)
	client.ResolveLFSFiles(ctx, items, target.Owner, target.Repo, target.Branch)

	cursor := 0
	if selectedFilePath != "" {
		cursor = findCursorByPath(items, selectedFilePath)
	}

	return loadResult{
		currentURL:   target,
		items:        items,
		hasFullTree:  false,
		folderSizes:  map[string]uint64{},
		cursor:       cursor,
		warning:      warning,
		sessionToken: sessionToken,
	}, nil
}

func fetchPreviewCmd(item gh.RepoItem, token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		content, err := fetchPreviewContent(ctx, gh.NewClient(token), item)
		if err != nil {
			return previewFailedMsg{path: item.Path, err: err}
		}

		return previewLoadedMsg{path: item.Path, content: content}
	}
}

func fetchPreviewContent(ctx context.Context, client *gh.Client, item gh.RepoItem) (string, error) {
	if item.IsLFS() {
		return "", fmt.Errorf("Git LFS files cannot be previewed here. Download the file to inspect it.")
	}
	if item.ActualSize() > maxPreviewBytes {
		return "", fmt.Errorf("this file is too large to preview in-app. Download it to inspect locally.")
	}

	downloadURL := item.ActualDownloadURL()
	if downloadURL == "" {
		return "", fmt.Errorf("no preview URL available for this file")
	}

	data, err := client.DownloadBinary(ctx, downloadURL)
	if err != nil {
		return "", err
	}
	if bytes.Contains(data, []byte{0}) {
		return "", fmt.Errorf("binary files are not previewed in the TUI")
	}

	content := string(data)
	if len([]rune(content)) <= maxPreviewChars {
		return content, nil
	}

	return string([]rune(content)[:maxPreviewChars]) + "\n\n[preview truncated after 40000 characters]", nil
}

func performDownloadCmd(request downloadRequest) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		client := gh.NewClient(request.token)
		items, err := collectDownloadItems(ctx, client, request.selected, request.fullTree, request.hasFullTree)
		if err != nil {
			return downloadFinishedMsg{err: err}
		}

		client.ResolveLFSFiles(ctx, items, request.currentURL.Owner, request.currentURL.Repo, request.currentURL.Branch)
		if len(items) == 0 {
			return downloadFinishedMsg{empty: true}
		}

		downloadDir, err := chooseDownloadDir(request)
		if err != nil {
			return downloadFinishedMsg{err: err}
		}

		downloader, err := download.New(downloadDir, request.token)
		if err != nil {
			return downloadFinishedMsg{err: err}
		}

		errors, err := downloader.DownloadItems(ctx, items, request.progress)
		return downloadFinishedMsg{
			downloadDir: downloadDir,
			errors:      errors,
			err:         err,
		}
	}
}

func chooseDownloadDir(request downloadRequest) (string, error) {
	baseDir := request.overridePath
	switch {
	case baseDir != "":
	case request.cwd:
		var err error
		baseDir, err = defaultDownloadDir()
		if err != nil {
			return "", err
		}
	case request.configPath != "":
		baseDir = request.configPath
	default:
		var err error
		baseDir, err = defaultDownloadDir()
		if err != nil {
			return "", err
		}
	}

	if request.noFolder {
		return baseDir, nil
	}

	return filepath.Join(baseDir, request.currentURL.Repo), nil
}
