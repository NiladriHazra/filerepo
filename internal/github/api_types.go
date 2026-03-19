package github

import "time"

// TargetKind identifies what the parsed GitHub URL points at.
type TargetKind string

const (
	TargetRepository  TargetKind = "repository"
	TargetCompare     TargetKind = "compare"
	TargetPullRequest TargetKind = "pull_request"
)

// RateLimitStatus captures the most recent GitHub API rate-limit headers.
type RateLimitStatus struct {
	Limit         int
	Remaining     int
	Used          int
	ResetAt       time.Time
	Authenticated bool
}

// RepoMetadata captures repository-level metadata shown in the UI.
type RepoMetadata struct {
	FullName      string
	Description   string
	DefaultBranch string
	Private       bool
	Archived      bool
	Language      string
	Stars         int
	Forks         int
	OpenIssues    int
	UpdatedAt     time.Time
	PushedAt      time.Time
	HTMLURL       string
}

// RepoRef is a branch or tag.
type RepoRef struct {
	Name       string
	Kind       string
	CommitSHA  string
	Protected  bool
	IsDefault  bool
	TarballURL string
	ZipballURL string
}

// Readme captures README metadata and decoded text content.
type Readme struct {
	Name        string
	Path        string
	HTMLURL     string
	DownloadURL string
	Content     string
}

// Release describes a GitHub release.
type Release struct {
	ID          int64
	Name        string
	TagName     string
	IsLatest    bool
	IsDraft     bool
	IsPre       bool
	PublishedAt time.Time
	Body        string
	Assets      []ReleaseAsset
}

// ReleaseAsset describes a release asset that can be downloaded.
type ReleaseAsset struct {
	ID            int64
	Name          string
	Size          uint64
	ContentType   string
	DownloadCount int
	UpdatedAt     time.Time
	DownloadURL   string
}

// CompareFile is a file entry returned by compare and PR endpoints.
type CompareFile struct {
	Filename    string `json:"filename"`
	Status      string `json:"status"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	Changes     int    `json:"changes"`
	RawURL      string `json:"raw_url"`
	BlobURL     string `json:"blob_url"`
	SHA         string `json:"sha"`
	PreviousSHA string `json:"previous_filename"`
	Patch       string `json:"patch"`
}

// Comparison captures changed files between two refs.
type Comparison struct {
	BaseRef   string
	HeadRef   string
	HTMLURL   string
	AheadBy   int
	BehindBy  int
	Files     []CompareFile
	CommitSHA string
}

// PullRequest captures summary information for a pull request.
type PullRequest struct {
	Number   int
	Title    string
	HTMLURL  string
	BaseRef  string
	HeadRef  string
	HeadSHA  string
	Files    []CompareFile
	Merged   bool
	State    string
	Draft    bool
	Body     string
	IsClosed bool
}
