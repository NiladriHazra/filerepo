package github

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// URL describes a GitHub repository URL and the selected tree path.
type URL struct {
	Owner  string
	Repo   string
	Branch string
	Path   string
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
	}

	if len(segments) >= 4 {
		switch segments[2] {
		case "tree", "blob":
			result.Branch = segments[3]
			if len(segments) > 4 {
				result.Path = strings.Join(segments[4:], "/")
			}
		}
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
