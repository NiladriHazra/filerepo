package github

import (
	"context"
	"fmt"
	"strings"
)

// LFSPointer describes the contents of a git-lfs pointer file.
type LFSPointer struct {
	OID  string
	Size uint64
}

type lfsBatchResponse struct {
	Objects []lfsResponseObject `json:"objects"`
}

type lfsResponseObject struct {
	Actions *lfsActions `json:"actions"`
}

type lfsActions struct {
	Download *lfsDownloadAction `json:"download"`
}

type lfsDownloadAction struct {
	Href string `json:"href"`
}

// ParseLFSPointer parses a text file into a git-lfs pointer if possible.
func ParseLFSPointer(content string) *LFSPointer {
	if !strings.HasPrefix(content, "version https://git-lfs.github.com/spec/v1") {
		return nil
	}

	var (
		oid     string
		size    uint64
		sizeSet bool
	)

	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.HasPrefix(line, "oid sha256:"):
			oid = strings.TrimPrefix(line, "oid sha256:")
		case strings.HasPrefix(line, "size "):
			if _, err := fmt.Sscanf(strings.TrimPrefix(line, "size "), "%d", &size); err == nil {
				sizeSet = true
			}
		}
	}

	if oid == "" || !sizeSet {
		return nil
	}

	return &LFSPointer{OID: oid, Size: size}
}

// ResolveLFSFiles augments visible file entries with their real LFS metadata.
func (c *Client) ResolveLFSFiles(ctx context.Context, items []RepoItem, owner, repo, branch string) {
	for index := range items {
		item := &items[index]
		switch {
		case !item.IsFile():
			continue
		case item.Size == nil:
			continue
		case *item.Size >= 1024:
			continue
		case item.DownloadURL == "":
			continue
		}

		content, err := c.FetchRawContent(ctx, item.DownloadURL)
		if err != nil {
			continue
		}

		pointer := ParseLFSPointer(content)
		if pointer == nil {
			continue
		}

		item.LFSOID = pointer.OID
		item.LFSSize = &pointer.Size

		lfsURL, err := c.GetLFSDownloadURL(ctx, owner, repo, pointer.OID, pointer.Size)
		if err == nil {
			item.LFSDownloadURL = lfsURL
			continue
		}

		item.LFSDownloadURL = fmt.Sprintf(
			"https://media.githubusercontent.com/media/%s/%s/%s/%s",
			owner,
			repo,
			branch,
			item.Path,
		)
	}
}
