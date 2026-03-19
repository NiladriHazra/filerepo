package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type repoAPIResponse struct {
	FullName         string    `json:"full_name"`
	Description      string    `json:"description"`
	DefaultBranch    string    `json:"default_branch"`
	Private          bool      `json:"private"`
	Archived         bool      `json:"archived"`
	Language         string    `json:"language"`
	StargazersCount  int       `json:"stargazers_count"`
	ForksCount       int       `json:"forks_count"`
	OpenIssuesCount  int       `json:"open_issues_count"`
	UpdatedAt        time.Time `json:"updated_at"`
	PushedAt         time.Time `json:"pushed_at"`
	HTMLURL          string    `json:"html_url"`
	DefaultBranchRef string    `json:"master_branch"`
}

type readmeAPIResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

type branchAPIResponse struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Commit    struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type tagAPIResponse struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
	TarballURL string `json:"tarball_url"`
	ZipballURL string `json:"zipball_url"`
}

type releaseAPIResponse struct {
	ID          int64              `json:"id"`
	Name        string             `json:"name"`
	TagName     string             `json:"tag_name"`
	Draft       bool               `json:"draft"`
	Prerelease  bool               `json:"prerelease"`
	PublishedAt time.Time          `json:"published_at"`
	Body        string             `json:"body"`
	Assets      []releaseAssetJSON `json:"assets"`
}

type releaseAssetJSON struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Size               uint64    `json:"size"`
	ContentType        string    `json:"content_type"`
	DownloadCount      int       `json:"download_count"`
	UpdatedAt          time.Time `json:"updated_at"`
	BrowserDownloadURL string    `json:"browser_download_url"`
}

type compareAPIResponse struct {
	HTMLURL         string        `json:"html_url"`
	AheadBy         int           `json:"ahead_by"`
	BehindBy        int           `json:"behind_by"`
	Files           []CompareFile `json:"files"`
	MergeBaseCommit struct {
		SHA string `json:"sha"`
	} `json:"merge_base_commit"`
}

type pullRequestAPIResponse struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
	Draft   bool   `json:"draft"`
	Merged  bool   `json:"merged"`
	Body    string `json:"body"`
	Base    struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
}

// FetchRepoMetadata loads repository-level metadata.
func (c *Client) FetchRepoMetadata(ctx context.Context, owner, repo string) (RepoMetadata, error) {
	var payload repoAPIResponse
	target := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	if err := c.FetchJSON(ctx, target, &payload); err != nil {
		return RepoMetadata{}, err
	}

	return RepoMetadata{
		FullName:      payload.FullName,
		Description:   payload.Description,
		DefaultBranch: nonEmpty(payload.DefaultBranch, payload.DefaultBranchRef),
		Private:       payload.Private,
		Archived:      payload.Archived,
		Language:      payload.Language,
		Stars:         payload.StargazersCount,
		Forks:         payload.ForksCount,
		OpenIssues:    payload.OpenIssuesCount,
		UpdatedAt:     payload.UpdatedAt,
		PushedAt:      payload.PushedAt,
		HTMLURL:       payload.HTMLURL,
	}, nil
}

// FetchREADME loads and decodes the repository README for the given ref.
func (c *Client) FetchREADME(ctx context.Context, owner, repo, ref string) (Readme, error) {
	var payload readmeAPIResponse
	target := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme?ref=%s", owner, repo, url.QueryEscape(ref))
	if err := c.FetchJSON(ctx, target, &payload); err != nil {
		return Readme{}, err
	}

	content := ""
	if strings.EqualFold(payload.Encoding, "base64") && payload.Content != "" {
		decoded, err := DecodeBase64Content(payload.Content)
		if err != nil {
			return Readme{}, err
		}
		content = decoded
	}

	return Readme{
		Name:        payload.Name,
		Path:        payload.Path,
		HTMLURL:     payload.HTMLURL,
		DownloadURL: payload.DownloadURL,
		Content:     content,
	}, nil
}

// FetchRefs loads a small list of branches and tags for the repository.
func (c *Client) FetchRefs(ctx context.Context, owner, repo, defaultBranch string) ([]RepoRef, error) {
	var branches []branchAPIResponse
	if err := c.FetchJSON(ctx, fmt.Sprintf("https://api.github.com/repos/%s/%s/branches?per_page=100", owner, repo), &branches); err != nil {
		return nil, err
	}

	var tags []tagAPIResponse
	if err := c.FetchJSON(ctx, fmt.Sprintf("https://api.github.com/repos/%s/%s/tags?per_page=100", owner, repo), &tags); err != nil {
		return nil, err
	}

	refs := make([]RepoRef, 0, len(branches)+len(tags))
	for _, branch := range branches {
		refs = append(refs, RepoRef{
			Name:      branch.Name,
			Kind:      "branch",
			CommitSHA: branch.Commit.SHA,
			Protected: branch.Protected,
			IsDefault: branch.Name == defaultBranch,
		})
	}
	for _, tag := range tags {
		refs = append(refs, RepoRef{
			Name:       tag.Name,
			Kind:       "tag",
			CommitSHA:  tag.Commit.SHA,
			TarballURL: tag.TarballURL,
			ZipballURL: tag.ZipballURL,
		})
	}
	return refs, nil
}

// FetchReleases loads repository releases and assets.
func (c *Client) FetchReleases(ctx context.Context, owner, repo string) ([]Release, error) {
	var payload []releaseAPIResponse
	target := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=20", owner, repo)
	if err := c.FetchJSON(ctx, target, &payload); err != nil {
		return nil, err
	}

	releases := make([]Release, 0, len(payload))
	for index, release := range payload {
		assets := make([]ReleaseAsset, 0, len(release.Assets))
		for _, asset := range release.Assets {
			assets = append(assets, ReleaseAsset{
				ID:            asset.ID,
				Name:          asset.Name,
				Size:          asset.Size,
				ContentType:   asset.ContentType,
				DownloadCount: asset.DownloadCount,
				UpdatedAt:     asset.UpdatedAt,
				DownloadURL:   asset.BrowserDownloadURL,
			})
		}

		releases = append(releases, Release{
			ID:          release.ID,
			Name:        release.Name,
			TagName:     release.TagName,
			IsLatest:    index == 0,
			IsDraft:     release.Draft,
			IsPre:       release.Prerelease,
			PublishedAt: release.PublishedAt,
			Body:        release.Body,
			Assets:      assets,
		})
	}

	return releases, nil
}

// FetchComparison loads changed files between two refs.
func (c *Client) FetchComparison(ctx context.Context, owner, repo, baseRef, headRef string) (Comparison, error) {
	var payload compareAPIResponse
	target := fmt.Sprintf("https://api.github.com/repos/%s/%s/compare/%s...%s", owner, repo, url.PathEscape(baseRef), url.PathEscape(headRef))
	if err := c.FetchJSON(ctx, target, &payload); err != nil {
		return Comparison{}, err
	}

	return Comparison{
		BaseRef:   baseRef,
		HeadRef:   headRef,
		HTMLURL:   payload.HTMLURL,
		AheadBy:   payload.AheadBy,
		BehindBy:  payload.BehindBy,
		Files:     payload.Files,
		CommitSHA: payload.MergeBaseCommit.SHA,
	}, nil
}

// FetchPullRequest loads pull-request metadata and changed files.
func (c *Client) FetchPullRequest(ctx context.Context, owner, repo string, number int) (PullRequest, error) {
	var payload pullRequestAPIResponse
	target := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, repo, number)
	if err := c.FetchJSON(ctx, target, &payload); err != nil {
		return PullRequest{}, err
	}

	var files []CompareFile
	filesTarget := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/files?per_page=100", owner, repo, number)
	if err := c.FetchJSON(ctx, filesTarget, &files); err != nil {
		return PullRequest{}, err
	}

	return PullRequest{
		Number:   payload.Number,
		Title:    payload.Title,
		HTMLURL:  payload.HTMLURL,
		BaseRef:  payload.Base.Ref,
		HeadRef:  payload.Head.Ref,
		HeadSHA:  payload.Head.SHA,
		Files:    files,
		Merged:   payload.Merged,
		State:    payload.State,
		Draft:    payload.Draft,
		Body:     payload.Body,
		IsClosed: payload.State == "closed",
	}, nil
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
