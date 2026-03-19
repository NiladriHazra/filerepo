package github

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// URL describes a GitHub repository URL and the selected tree path.
type URL struct {
	Owner       string
	Repo        string
	Branch      string
	Path        string
	Kind        TargetKind
	CompareBase string
	CompareHead string
	PullNumber  int
}

// ParseURL parses a github.com repository, tree, or blob URL.
func ParseURL(raw string) (URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return URL{}, fmt.Errorf("invalid URL format: %w", err)
	}

	if parsed.Host != "github.com" {
		return URL{}, fmt.Errorf("not a GitHub URL")
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 2 {
		return URL{}, fmt.Errorf("URL must contain owner and repository")
	}

	result := URL{
		Owner:  segments[0],
		Repo:   segments[1],
		Branch: "main",
		Kind:   TargetRepository,
	}

	if len(segments) < 3 {
		return result, nil
	}

	switch segments[2] {
	case "tree", "blob":
		if len(segments) < 4 {
			return URL{}, fmt.Errorf("missing ref name in GitHub URL")
		}
		result.Branch = segments[3]
		if len(segments) > 4 {
			result.Path = strings.Join(segments[4:], "/")
		}
	case "commit":
		if len(segments) < 4 {
			return URL{}, fmt.Errorf("missing commit SHA in GitHub URL")
		}
		result.Branch = segments[3]
	case "compare":
		if len(segments) < 4 {
			return URL{}, fmt.Errorf("missing compare refs in GitHub URL")
		}
		base, head, ok := strings.Cut(segments[3], "...")
		if !ok || base == "" || head == "" {
			return URL{}, fmt.Errorf("invalid compare URL")
		}
		result.Kind = TargetCompare
		result.CompareBase = base
		result.CompareHead = head
		result.Branch = head
	case "pull":
		if len(segments) < 4 {
			return URL{}, fmt.Errorf("missing pull request number in GitHub URL")
		}
		number := 0
		if _, err := fmt.Sscanf(segments[3], "%d", &number); err != nil || number <= 0 {
			return URL{}, fmt.Errorf("invalid pull request number")
		}
		result.Kind = TargetPullRequest
		result.PullNumber = number
	}

	return result, nil
}

// APIURL returns the GitHub contents API URL for the selected repository path.
func (u URL) APIURL() string {
	base := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents", u.Owner, u.Repo)
	if u.Path == "" {
		return fmt.Sprintf("%s?ref=%s", base, url.QueryEscape(u.Branch))
	}
	return fmt.Sprintf("%s/%s?ref=%s", base, u.Path, url.QueryEscape(u.Branch))
}

// RepoURL returns the canonical repository URL.
func (u URL) RepoURL() string {
	return fmt.Sprintf("https://github.com/%s/%s", u.Owner, u.Repo)
}

// WebURL returns the canonical GitHub URL for the target.
func (u URL) WebURL() string {
	switch u.Kind {
	case TargetCompare:
		return fmt.Sprintf("%s/compare/%s...%s", u.RepoURL(), u.CompareBase, u.CompareHead)
	case TargetPullRequest:
		return fmt.Sprintf("%s/pull/%d", u.RepoURL(), u.PullNumber)
	default:
		if u.Path == "" && u.Branch == "main" {
			return u.RepoURL()
		}
		if u.Path == "" {
			return fmt.Sprintf("%s/tree/%s", u.RepoURL(), u.Branch)
		}
		return fmt.Sprintf("%s/tree/%s/%s", u.RepoURL(), u.Branch, u.Path)
	}
}

// GetLocalGitRemote resolves the current git origin into an https GitHub URL.
func GetLocalGitRemote() string {
	output, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}

	raw := strings.TrimSpace(string(output))
	switch {
	case raw == "":
		return ""
	case strings.HasPrefix(raw, "git@github.com:"):
		path := strings.TrimPrefix(raw, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		return "https://github.com/" + path
	default:
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host != "github.com" {
			return ""
		}
		return "https://github.com" + strings.TrimSuffix(parsed.Path, ".git")
	}
}
