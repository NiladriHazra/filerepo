package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NiladriHazra/filerepo/internal/config"
)

type envelope[T any] struct {
	SavedAt time.Time `json:"saved_at"`
	Value   T         `json:"value"`
}

// Load reads a cached JSON value when it is still fresh enough.
func Load[T any](name string, maxAge time.Duration) (T, bool, error) {
	var zero T
	path, err := filePath(name)
	if err != nil {
		return zero, false, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("read cache file: %w", err)
	}

	var cached envelope[T]
	if err := json.Unmarshal(data, &cached); err != nil {
		return zero, false, fmt.Errorf("parse cache file: %w", err)
	}

	if maxAge > 0 && time.Since(cached.SavedAt) > maxAge {
		return zero, false, nil
	}

	return cached.Value, true, nil
}

// Save persists a JSON cache entry.
func Save[T any](name string, value T) error {
	path, err := filePath(name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	payload, err := json.MarshalIndent(envelope[T]{
		SavedAt: time.Now().UTC(),
		Value:   value,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode cache entry: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	return nil
}

func filePath(name string) (string, error) {
	cacheDir, err := config.CachePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, sanitize(name)+".json"), nil
}

func sanitize(name string) string {
	var builder strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	return builder.String()
}
