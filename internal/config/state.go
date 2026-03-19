package config

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	// DefaultProfileName is the profile filerepo uses when no explicit profile is selected.
	DefaultProfileName = "default"
	maxRecentRepos     = 8
	defaultCacheTTL    = 30 * time.Minute
)

// Profile stores GitHub credentials for a named profile.
type Profile struct {
	GitHubToken string `json:"github_token,omitempty"`
}

// RepoEntry stores a recent or favorite repository URL.
type RepoEntry struct {
	URL        string    `json:"url"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
}

// CacheSettings controls on-disk API caching.
type CacheSettings struct {
	Enabled    bool `json:"enabled"`
	TTLMinutes int  `json:"ttl_minutes,omitempty"`
}

// ActiveToken returns the token for the active profile.
func (c Config) ActiveToken() string {
	profile := c.activeProfileName()
	if details, ok := c.Profiles[profile]; ok && details.GitHubToken != "" {
		return details.GitHubToken
	}
	return c.GitHubToken
}

// ActiveProfileName returns the active profile name.
func (c Config) ActiveProfileName() string {
	return c.activeProfileName()
}

// ProfileNames returns profile names in stable order with the active profile first.
func (c Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	if len(names) == 0 {
		return []string{c.activeProfileName()}
	}
	slices.Sort(names)
	active := c.activeProfileName()
	for index, name := range names {
		if name != active {
			continue
		}
		names[0], names[index] = names[index], names[0]
		break
	}
	return names
}

// SetActiveProfile switches the active profile.
func (c *Config) SetActiveProfile(name string) error {
	name = normalizeProfileName(name)
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	c.normalize()
	if _, ok := c.Profiles[name]; !ok {
		c.Profiles[name] = Profile{}
	}
	c.ActiveProfile = name
	c.syncLegacyToken()
	return nil
}

// SetProfileToken saves a token on the named profile.
func (c *Config) SetProfileToken(name, token string) error {
	name = normalizeProfileName(name)
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	c.normalize()
	profile := c.Profiles[name]
	profile.GitHubToken = strings.TrimSpace(token)
	c.Profiles[name] = profile
	if c.ActiveProfile == "" {
		c.ActiveProfile = name
	}
	c.syncLegacyToken()
	return nil
}

// UnsetProfileToken clears a token on the named profile.
func (c *Config) UnsetProfileToken(name string) error {
	name = normalizeProfileName(name)
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	c.normalize()
	profile := c.Profiles[name]
	profile.GitHubToken = ""
	c.Profiles[name] = profile
	c.syncLegacyToken()
	return nil
}

// AddRecentRepo records a recently opened repository URL.
func (c *Config) AddRecentRepo(rawURL string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return
	}

	c.normalize()
	entries := make([]RepoEntry, 0, len(c.RecentRepos)+1)
	entries = append(entries, RepoEntry{
		URL:        rawURL,
		LastUsedAt: time.Now().UTC(),
	})

	for _, entry := range c.RecentRepos {
		if entry.URL == "" || entry.URL == rawURL {
			continue
		}
		entries = append(entries, entry)
		if len(entries) >= maxRecentRepos {
			break
		}
	}

	c.RecentRepos = entries
}

// AddFavorite stores a pinned repository URL if it is not already present.
func (c *Config) AddFavorite(rawURL string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return
	}

	c.normalize()
	for _, entry := range c.Favorites {
		if entry.URL == rawURL {
			return
		}
	}

	c.Favorites = append(c.Favorites, RepoEntry{
		URL:        rawURL,
		LastUsedAt: time.Now().UTC(),
	})
	slices.SortFunc(c.Favorites, func(a, b RepoEntry) int {
		return strings.Compare(a.URL, b.URL)
	})
}

// RemoveFavorite removes a pinned repository URL.
func (c *Config) RemoveFavorite(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}

	index := slices.IndexFunc(c.Favorites, func(entry RepoEntry) bool {
		return entry.URL == rawURL
	})
	if index < 0 {
		return false
	}

	c.Favorites = append(c.Favorites[:index], c.Favorites[index+1:]...)
	return true
}

// IsFavorite reports whether the repository is pinned.
func (c Config) IsFavorite(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	return slices.ContainsFunc(c.Favorites, func(entry RepoEntry) bool {
		return entry.URL == rawURL
	})
}

// ClearRecentRepos removes the recent repository history.
func (c *Config) ClearRecentRepos() {
	c.RecentRepos = nil
}

// CacheTTL returns the configured cache TTL or a sensible default.
func (c Config) CacheTTL() time.Duration {
	switch minutes := c.Cache.TTLMinutes; {
	case minutes <= 0:
		return defaultCacheTTL
	default:
		return time.Duration(minutes) * time.Minute
	}
}

func (c *Config) normalize() {
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	if c.ActiveProfile == "" {
		c.ActiveProfile = DefaultProfileName
	}
	if c.GitHubToken != "" {
		profile := c.Profiles[c.ActiveProfile]
		if profile.GitHubToken == "" {
			profile.GitHubToken = c.GitHubToken
			c.Profiles[c.ActiveProfile] = profile
		}
	}
	c.Cache.Enabled = true
	c.RecentRepos = dedupeEntries(c.RecentRepos, maxRecentRepos)
	c.Favorites = dedupeEntries(c.Favorites, 0)
	c.syncLegacyToken()
}

func (c *Config) syncLegacyToken() {
	c.GitHubToken = ""
	active := c.activeProfileName()
	if profile, ok := c.Profiles[active]; ok {
		c.GitHubToken = profile.GitHubToken
	}
}

func (c Config) activeProfileName() string {
	if strings.TrimSpace(c.ActiveProfile) == "" {
		return DefaultProfileName
	}
	return normalizeProfileName(c.ActiveProfile)
}

func dedupeEntries(entries []RepoEntry, limit int) []RepoEntry {
	seen := map[string]struct{}{}
	result := make([]RepoEntry, 0, len(entries))

	for _, entry := range entries {
		entry.URL = strings.TrimSpace(entry.URL)
		if entry.URL == "" {
			continue
		}
		if _, ok := seen[entry.URL]; ok {
			continue
		}
		seen[entry.URL] = struct{}{}
		result = append(result, entry)
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result
}

func normalizeProfileName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	return name
}
