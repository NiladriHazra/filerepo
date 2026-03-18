package github

// RepoItem is a file or directory returned by the GitHub contents or tree APIs.
type RepoItem struct {
	Name           string  `json:"name"`
	ItemType       string  `json:"type"`
	Path           string  `json:"path"`
	DownloadURL    string  `json:"download_url"`
	URL            string  `json:"url"`
	Size           *uint64 `json:"size"`
	Selected       bool    `json:"-"`
	LFSOID         string  `json:"-"`
	LFSSize        *uint64 `json:"-"`
	LFSDownloadURL string  `json:"-"`
}

// IsDir reports whether the item is a directory.
func (r RepoItem) IsDir() bool {
	return r.ItemType == "dir"
}

// IsFile reports whether the item is a file.
func (r RepoItem) IsFile() bool {
	return r.ItemType == "file"
}

// IsLFS reports whether the item is backed by git-lfs.
func (r RepoItem) IsLFS() bool {
	return r.LFSOID != ""
}

// ActualSize returns the LFS size when present, otherwise the GitHub size.
func (r RepoItem) ActualSize() uint64 {
	switch {
	case r.LFSSize != nil:
		return *r.LFSSize
	case r.Size != nil:
		return *r.Size
	default:
		return 0
	}
}

// ActualDownloadURL returns the direct LFS download URL when available.
func (r RepoItem) ActualDownloadURL() string {
	if r.LFSDownloadURL != "" {
		return r.LFSDownloadURL
	}
	return r.DownloadURL
}

// GitTreeResponse is the response payload from the recursive tree API.
type GitTreeResponse struct {
	Tree      []GitTreeEntry `json:"tree"`
	Truncated bool           `json:"truncated"`
}

// GitTreeEntry is a single recursive tree entry.
type GitTreeEntry struct {
	Path      string  `json:"path"`
	Mode      string  `json:"mode"`
	EntryType string  `json:"type"`
	Size      *uint64 `json:"size"`
	SHA       string  `json:"sha"`
	URL       string  `json:"url"`
}
