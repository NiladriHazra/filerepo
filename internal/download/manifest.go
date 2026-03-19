package download

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteManifest persists a download manifest alongside the downloaded output.
func WriteManifest(baseDir string, manifest Manifest) (string, error) {
	if manifest.GeneratedAt.IsZero() {
		manifest.GeneratedAt = time.Now().UTC()
	}

	name := fmt.Sprintf("filerepo-manifest-%s.json", manifest.GeneratedAt.Format("20060102-150405"))
	path := filepath.Join(baseDir, name)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write manifest: %w", err)
	}

	return path, nil
}
