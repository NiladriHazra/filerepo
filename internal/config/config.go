package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config stores filerepo's persisted token and download directory.
type Config struct {
	GitHubToken   string             `json:"github_token,omitempty"`
	DownloadPath  string             `json:"download_path,omitempty"`
	ActiveProfile string             `json:"active_profile,omitempty"`
	Profiles      map[string]Profile `json:"profiles,omitempty"`
	RecentRepos   []RepoEntry        `json:"recent_repos,omitempty"`
	Favorites     []RepoEntry        `json:"favorites,omitempty"`
	Cache         CacheSettings      `json:"cache,omitempty"`
}

// Load reads the filerepo config file from the user config directory.
func Load() (Config, error) {
	configPath, err := path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		var cfg Config
		cfg.normalize()
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}
	cfg.normalize()

	return cfg, nil
}

// Save persists the filerepo configuration to disk.
func Save(cfg Config) error {
	cfg.normalize()

	configPath, err := path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// ValidatePath ensures the configured download path exists and is writable.
func ValidatePath(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("path does not exist: %s", dir)
		}
		return fmt.Errorf("stat path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dir)
	}

	testFile, err := os.CreateTemp(dir, ".filerepo-write-check-*")
	if err != nil {
		return fmt.Errorf("path is not writable: %s", dir)
	}
	testFile.Close()
	if err := os.Remove(testFile.Name()); err != nil {
		return fmt.Errorf("cleanup write check: %w", err)
	}

	return nil
}

// Path returns the on-disk config file path.
func Path() (string, error) {
	return path()
}

// CachePath returns the on-disk cache directory path.
func CachePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}

	return filepath.Join(base, "filerepo", "cache"), nil
}

func path() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}

	return filepath.Join(base, "filerepo", "config.json"), nil
}
