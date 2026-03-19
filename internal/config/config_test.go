package config

import "testing"

func TestNormalizeLegacyTokenIntoDefaultProfile(t *testing.T) {
	t.Parallel()

	cfg := Config{
		GitHubToken: "ghp_legacy",
	}
	cfg.normalize()

	if got, want := cfg.ActiveProfileName(), DefaultProfileName; got != want {
		t.Fatalf("active profile mismatch: got %q want %q", got, want)
	}

	if got, want := cfg.ActiveToken(), "ghp_legacy"; got != want {
		t.Fatalf("active token mismatch: got %q want %q", got, want)
	}
}

func TestAddRecentRepoDedupesAndCaps(t *testing.T) {
	t.Parallel()

	var cfg Config
	for _, rawURL := range []string{
		"https://github.com/owner/repo1",
		"https://github.com/owner/repo2",
		"https://github.com/owner/repo1",
		"https://github.com/owner/repo3",
		"https://github.com/owner/repo4",
		"https://github.com/owner/repo5",
		"https://github.com/owner/repo6",
		"https://github.com/owner/repo7",
		"https://github.com/owner/repo8",
		"https://github.com/owner/repo9",
	} {
		cfg.AddRecentRepo(rawURL)
	}

	if got, want := len(cfg.RecentRepos), maxRecentRepos; got != want {
		t.Fatalf("recent repo count mismatch: got %d want %d", got, want)
	}

	if got, want := cfg.RecentRepos[0].URL, "https://github.com/owner/repo9"; got != want {
		t.Fatalf("recent repo head mismatch: got %q want %q", got, want)
	}

	for index := 1; index < len(cfg.RecentRepos); index++ {
		if cfg.RecentRepos[index].URL == cfg.RecentRepos[0].URL {
			t.Fatalf("duplicate recent repo found: %q", cfg.RecentRepos[index].URL)
		}
	}
}

func TestFavoriteRoundTrip(t *testing.T) {
	t.Parallel()

	var cfg Config
	rawURL := "https://github.com/owner/repo"

	cfg.AddFavorite(rawURL)
	if !cfg.IsFavorite(rawURL) {
		t.Fatalf("expected favorite %q to be present", rawURL)
	}

	if !cfg.RemoveFavorite(rawURL) {
		t.Fatalf("expected favorite %q to be removable", rawURL)
	}
	if cfg.IsFavorite(rawURL) {
		t.Fatalf("expected favorite %q to be removed", rawURL)
	}
}
