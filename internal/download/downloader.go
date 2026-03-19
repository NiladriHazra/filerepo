package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	options  Options
}

// New creates a downloader rooted at basePath.
func New(basePath string, token string, options Options) (*Downloader, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("create download directory: %w", err)
	}
	if options.ConflictStrategy == "" {
		options.ConflictStrategy = ConflictSkip
	}
	if options.OutputMode == "" {
		options.OutputMode = OutputFiles
	}

	return &Downloader{
		client:   gh.NewClient(token),
		basePath: basePath,
		options:  options,
	}, nil
}

// DownloadItems downloads the provided items with bounded concurrency.
func (d *Downloader) DownloadItems(ctx context.Context, items []gh.RepoItem, progress *Progress) (Result, []string, error) {
	if progress == nil {
		return Result{}, nil, errors.New("download progress is required")
	}

	switch d.options.OutputMode {
	case OutputZip:
		path, err := d.downloadZipArchive(ctx, items, progress)
		return Result{OutputPath: path}, nil, err
	case OutputTarGz:
		path, err := d.downloadTarGzArchive(ctx, items, progress)
		return Result{OutputPath: path}, nil, err
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
	return Result{OutputPath: d.basePath}, errs, nil
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

	finalPath, err := d.writeContent(destPath, content)
	if err != nil {
		return fmt.Errorf("write file %s: %w", destPath, err)
	}
	progress.SetCurrentFile(filepath.Base(finalPath))

	return nil
}

func (d *Downloader) writeContent(destPath string, content []byte) (string, error) {
	info, err := os.Stat(destPath)
	switch {
	case err == nil && info.IsDir():
		return "", fmt.Errorf("destination is a directory: %s", destPath)
	case err == nil:
		return d.resolveExistingFile(destPath, content)
	case errors.Is(err, os.ErrNotExist):
		return destPath, os.WriteFile(destPath, content, 0o644)
	case err != nil:
		return "", err
	}

	return destPath, os.WriteFile(destPath, content, 0o644)
}

func (d *Downloader) resolveExistingFile(destPath string, content []byte) (string, error) {
	switch d.options.ConflictStrategy {
	case ConflictOverwrite:
		return destPath, os.WriteFile(destPath, content, 0o644)
	case ConflictRename:
		nextPath := nextAvailablePath(destPath)
		return nextPath, os.WriteFile(nextPath, content, 0o644)
	case ConflictResume:
		existing, err := os.ReadFile(destPath)
		if err != nil {
			return "", err
		}
		switch {
		case len(existing) >= len(content):
			return destPath, nil
		case len(existing) > 0 && !bytesEqual(content[:len(existing)], existing):
			return "", fmt.Errorf("cannot resume %s because existing content differs", destPath)
		default:
			file, err := os.OpenFile(destPath, os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return "", err
			}
			defer file.Close()
			if _, err := file.Write(content[len(existing):]); err != nil {
				return "", err
			}
			return destPath, nil
		}
	default:
		return destPath, nil
	}
}

func (d *Downloader) downloadZipArchive(ctx context.Context, items []gh.RepoItem, progress *Progress) (string, error) {
	outputPath, err := d.archivePath(".zip")
	if err != nil {
		return "", err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create zip archive: %w", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for _, item := range items {
		progress.SetCurrentFile(item.Path)
		content, err := d.client.DownloadBinary(ctx, item.ActualDownloadURL())
		if err != nil {
			writer.Close()
			return "", err
		}

		entry, err := writer.Create(filepath.ToSlash(item.Path))
		if err != nil {
			writer.Close()
			return "", err
		}
		if _, err := entry.Write(content); err != nil {
			writer.Close()
			return "", err
		}
		progress.Advance()
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("finalize zip archive: %w", err)
	}

	return outputPath, nil
}

func (d *Downloader) downloadTarGzArchive(ctx context.Context, items []gh.RepoItem, progress *Progress) (string, error) {
	outputPath, err := d.archivePath(".tar.gz")
	if err != nil {
		return "", err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create tar.gz archive: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, item := range items {
		progress.SetCurrentFile(item.Path)
		content, err := d.client.DownloadBinary(ctx, item.ActualDownloadURL())
		if err != nil {
			return "", err
		}

		header := &tar.Header{
			Name: filepath.ToSlash(item.Path),
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return "", err
		}
		if _, err := tarWriter.Write(content); err != nil {
			return "", err
		}
		progress.Advance()
	}

	if err := tarWriter.Close(); err != nil {
		return "", fmt.Errorf("finalize tar archive: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return "", fmt.Errorf("finalize gzip stream: %w", err)
	}

	return outputPath, nil
}

func (d *Downloader) archivePath(ext string) (string, error) {
	base := filepath.Join(d.basePath, "filerepo-selection"+ext)
	info, err := os.Stat(base)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return base, nil
	case err != nil:
		return "", err
	case info.IsDir():
		return "", fmt.Errorf("archive destination is a directory: %s", base)
	}

	switch d.options.ConflictStrategy {
	case ConflictOverwrite, ConflictResume:
		return base, nil
	case ConflictRename:
		return nextAvailablePath(base), nil
	default:
		return "", fmt.Errorf("archive already exists: %s", base)
	}
}

func nextAvailablePath(path string) string {
	ext := filepath.Ext(path)
	name := strings.TrimSuffix(path, ext)
	if strings.HasSuffix(path, ".tar.gz") {
		ext = ".tar.gz"
		name = strings.TrimSuffix(path, ext)
	}
	for index := 1; ; index++ {
		next := fmt.Sprintf("%s (%d)%s", name, index, ext)
		if _, err := os.Stat(next); errors.Is(err, os.ErrNotExist) {
			return next
		}
	}
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
