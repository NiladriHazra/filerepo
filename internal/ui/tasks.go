package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/NiladriHazra/filerepo/internal/cache"
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
	repoURL      string
	metadata     gh.RepoMetadata
	refs         []gh.RepoRef
	readme       gh.Readme
	releases     []gh.Release
	rateLimit    gh.RateLimitStatus
}

type downloadRequest struct {
	currentURL   gh.URL
	repoURL      string
	selected     []gh.RepoItem
	fullTree     []gh.RepoItem
	hasFullTree  bool
	token        string
	configPath   string
	overridePath string
	cwd          bool
	noFolder     bool
	conflict     download.ConflictStrategy
	outputMode   download.OutputMode
	progress     *download.Progress
}

func loadRepoCmd(rawURL, token string, cacheEnabled bool, cacheTTL time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		parsed, err := gh.ParseURL(stringsTrimmed(rawURL))
		if err != nil {
			return repoLoadFailedMsg{err: fmt.Errorf("invalid URL: %w", err)}
		}

		result, err := loadRepository(ctx, parsed, token, cacheEnabled, cacheTTL)
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
			repoURL:      result.repoURL,
			metadata:     result.metadata,
			refs:         result.refs,
			readme:       result.readme,
			releases:     result.releases,
			rateLimit:    result.rateLimit,
		}
	}
}

func loadRepository(ctx context.Context, target gh.URL, token string, cacheEnabled bool, cacheTTL time.Duration) (loadResult, error) {
	switch target.Kind {
	case gh.TargetCompare:
		return loadCompareView(ctx, target, token)
	case gh.TargetPullRequest:
		return loadPullRequestView(ctx, target, token)
	}

	if cacheEnabled {
		if cached, ok, err := loadCachedRepository(target, cacheTTL); err == nil && ok {
			return snapshotToLoadResult(target, cached, token), nil
		}
	}

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
		extras := loadRepositoryExtras(ctx, client, target.Owner, target.Repo, target.Branch)
		result := loadResult{
			currentURL:   target,
			items:        currentItems,
			fullTree:     allItems,
			hasFullTree:  true,
			folderSizes:  calculateFolderSizes(allItems),
			cursor:       cursor,
			warning:      warning,
			sessionToken: sessionToken,
			repoURL:      target.WebURL(),
			metadata:     extras.metadata,
			refs:         extras.refs,
			readme:       extras.readme,
			releases:     extras.releases,
			rateLimit:    client.Status(),
		}
		if cacheEnabled {
			_ = saveCachedRepository(result)
		}
		return result, nil
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

	extras := loadRepositoryExtras(ctx, client, target.Owner, target.Repo, target.Branch)
	return loadResult{
		currentURL:   target,
		items:        items,
		hasFullTree:  false,
		folderSizes:  map[string]uint64{},
		cursor:       cursor,
		warning:      warning,
		sessionToken: sessionToken,
		repoURL:      target.WebURL(),
		metadata:     extras.metadata,
		refs:         extras.refs,
		readme:       extras.readme,
		releases:     extras.releases,
		rateLimit:    client.Status(),
	}, nil
}

func loadCompareView(ctx context.Context, target gh.URL, token string) (loadResult, error) {
	client := gh.NewClient(token)
	comparison, err := client.FetchComparison(ctx, target.Owner, target.Repo, target.CompareBase, target.CompareHead)
	if err != nil {
		return loadResult{}, err
	}

	items := mapCompareFilesToItems(comparison.Files)
	target.Branch = comparison.HeadRef
	extras := loadRepositoryExtras(ctx, client, target.Owner, target.Repo, comparison.HeadRef)
	return loadResult{
		currentURL:   target,
		items:        items,
		fullTree:     items,
		hasFullTree:  true,
		folderSizes:  map[string]uint64{},
		cursor:       0,
		sessionToken: token,
		repoURL:      target.WebURL(),
		metadata:     extras.metadata,
		refs:         extras.refs,
		readme:       extras.readme,
		releases:     extras.releases,
		rateLimit:    client.Status(),
	}, nil
}

func loadPullRequestView(ctx context.Context, target gh.URL, token string) (loadResult, error) {
	client := gh.NewClient(token)
	pullRequest, err := client.FetchPullRequest(ctx, target.Owner, target.Repo, target.PullNumber)
	if err != nil {
		return loadResult{}, err
	}

	items := mapCompareFilesToItems(pullRequest.Files)
	target.Branch = pullRequest.HeadRef
	extras := loadRepositoryExtras(ctx, client, target.Owner, target.Repo, pullRequest.HeadRef)
	return loadResult{
		currentURL:   target,
		items:        items,
		fullTree:     items,
		hasFullTree:  true,
		folderSizes:  map[string]uint64{},
		cursor:       0,
		warning:      fmt.Sprintf("Pull request #%d: %s", pullRequest.Number, pullRequest.Title),
		sessionToken: token,
		repoURL:      target.WebURL(),
		metadata:     extras.metadata,
		refs:         extras.refs,
		readme:       extras.readme,
		releases:     extras.releases,
		rateLimit:    client.Status(),
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

		downloader, err := download.New(downloadDir, request.token, download.Options{
			ConflictStrategy: request.conflict,
			OutputMode:       request.outputMode,
		})
		if err != nil {
			return downloadFinishedMsg{err: err}
		}

		result, errors, err := downloader.DownloadItems(ctx, items, request.progress)
		manifestPath := ""
		if err == nil {
			manifestPath, _ = download.WriteManifest(downloadDir, download.Manifest{
				RepositoryURL:    request.repoURL,
				Ref:              request.currentURL.Branch,
				OutputMode:       request.outputMode,
				ConflictStrategy: request.conflict,
				OutputPath:       result.OutputPath,
				SelectedPaths:    selectedPaths(items),
			})
		}

		return downloadFinishedMsg{
			downloadDir: downloadDir,
			outputPath:  result.OutputPath,
			manifest:    manifestPath,
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

type repoExtras struct {
	metadata gh.RepoMetadata
	refs     []gh.RepoRef
	readme   gh.Readme
	releases []gh.Release
}

func loadRepositoryExtras(ctx context.Context, client *gh.Client, owner, repo, branch string) repoExtras {
	var extras repoExtras

	metadata, err := client.FetchRepoMetadata(ctx, owner, repo)
	if err == nil {
		extras.metadata = metadata
		extras.refs, _ = client.FetchRefs(ctx, owner, repo, metadata.DefaultBranch)
	} else {
		extras.refs, _ = client.FetchRefs(ctx, owner, repo, branch)
	}

	readme, err := client.FetchREADME(ctx, owner, repo, branch)
	if err == nil {
		extras.readme = readme
	}

	releases, err := client.FetchReleases(ctx, owner, repo)
	if err == nil {
		extras.releases = releases
	}

	return extras
}

type repoSnapshot struct {
	CurrentURL  gh.URL
	FullTree    []gh.RepoItem
	FolderSizes map[string]uint64
	Metadata    gh.RepoMetadata
	Refs        []gh.RepoRef
	Readme      gh.Readme
	Releases    []gh.Release
}

func loadCachedRepository(target gh.URL, cacheTTL time.Duration) (repoSnapshot, bool, error) {
	key := repoCacheKey(target)
	return cache.Load[repoSnapshot](key, cacheTTL)
}

func saveCachedRepository(result loadResult) error {
	if !result.hasFullTree {
		return nil
	}

	snapshot := repoSnapshot{
		CurrentURL:  result.currentURL,
		FullTree:    result.fullTree,
		FolderSizes: result.folderSizes,
		Metadata:    result.metadata,
		Refs:        result.refs,
		Readme:      result.readme,
		Releases:    result.releases,
	}
	return cache.Save(repoCacheKey(result.currentURL), snapshot)
}

func snapshotToLoadResult(target gh.URL, snapshot repoSnapshot, token string) loadResult {
	browsePath, cursorPath := resolveRequestedView(snapshot.FullTree, target.Path)
	items := repoItemsForPath(snapshot.FullTree, browsePath)
	cursor := 0
	if cursorPath != "" {
		cursor = findCursorByPath(items, cursorPath)
	}

	target.Path = browsePath
	return loadResult{
		currentURL:   target,
		items:        items,
		fullTree:     snapshot.FullTree,
		hasFullTree:  true,
		folderSizes:  snapshot.FolderSizes,
		cursor:       cursor,
		warning:      "Loaded repository tree from cache.",
		sessionToken: token,
		repoURL:      target.WebURL(),
		metadata:     snapshot.Metadata,
		refs:         snapshot.Refs,
		readme:       snapshot.Readme,
		releases:     snapshot.Releases,
	}
}

func repoCacheKey(target gh.URL) string {
	return fmt.Sprintf("repo_%s_%s_%s", target.Owner, target.Repo, target.Branch)
}

func selectedPaths(items []gh.RepoItem) []string {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		paths = append(paths, item.Path)
	}
	return paths
}
