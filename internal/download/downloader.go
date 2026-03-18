package download

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	gh "github.com/NiladriHazra/filerepo/internal/github"
)

const maxConcurrentDownloads = 8

// Progress tracks download progress for the UI.
type Progress struct {
	Total int

	completed   atomic.Int64
	currentFile string
	mu          sync.Mutex
}

// Completed returns the number of finished downloads.
func (p *Progress) Completed() int {
	return int(p.completed.Load())
}

// Advance increments the number of completed downloads.
func (p *Progress) Advance() {
	p.completed.Add(1)
}

// SetCurrentFile records the path currently being downloaded.
func (p *Progress) SetCurrentFile(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentFile = path
}

// CurrentFile returns the path currently being downloaded.
func (p *Progress) CurrentFile() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentFile
}

// Downloader writes GitHub files into a destination directory.
type Downloader struct {
	client   *gh.Client
	basePath string
}

// New creates a downloader rooted at basePath.
func New(basePath string, token string) (*Downloader, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("create download directory: %w", err)
	}

	return &Downloader{
		client:   gh.NewClient(token),
		basePath: basePath,
	}, nil
}

// DownloadItems downloads the provided items with bounded concurrency.
func (d *Downloader) DownloadItems(ctx context.Context, items []gh.RepoItem, progress *Progress) ([]string, error) {
	if progress == nil {
		return nil, errors.New("download progress is required")
	}

	var (
		wg    sync.WaitGroup
		errMu sync.Mutex
		errs  []string
		sem   = make(chan struct{}, maxConcurrentDownloads)
	)

	for _, item := range items {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				errMu.Lock()
				errs = append(errs, fmt.Sprintf("download cancelled for %s", item.Name))
				errMu.Unlock()
				return
			}
			defer func() { <-sem }()

			destPath := filepath.Join(d.basePath, filepath.FromSlash(item.Name))
			if err := d.downloadFile(ctx, item, destPath, progress); err != nil {
				errMu.Lock()
				errs = append(errs, fmt.Sprintf("failed to download %s: %v", item.Name, err))
				errMu.Unlock()
			}
			progress.Advance()
		}()
	}

	wg.Wait()
	return errs, nil
}

func (d *Downloader) downloadFile(ctx context.Context, item gh.RepoItem, destPath string, progress *Progress) error {
	downloadURL := item.ActualDownloadURL()
	if downloadURL == "" {
		return errors.New("no download URL for file")
	}

	progress.SetCurrentFile(item.Name)

	content, err := d.client.DownloadBinary(ctx, downloadURL)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := os.WriteFile(destPath, content, 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", destPath, err)
	}

	return nil
}
