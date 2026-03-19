package ui

import (
	"time"

	gh "github.com/NiladriHazra/filerepo/internal/github"
)

type tickMsg time.Time

type repoLoadedMsg struct {
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

type repoLoadFailedMsg struct {
	err error
}

type previewLoadedMsg struct {
	path    string
	content string
}

type previewFailedMsg struct {
	path string
	err  error
}

type downloadFinishedMsg struct {
	downloadDir string
	outputPath  string
	manifest    string
	errors      []string
	empty       bool
	err         error
}
