package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NiladriHazra/filerepo/internal/config"
	"github.com/NiladriHazra/filerepo/internal/download"
	gh "github.com/NiladriHazra/filerepo/internal/github"
)

func TestInputViewIncludesSavedPanel(t *testing.T) {
	t.Parallel()

	cfg := config.Config{}
	cfg.AddFavorite("https://github.com/owner/repo")
	cfg.AddRecentRepo("https://github.com/owner/recent")

	model := newModel("", cfg, RunOptions{ActiveProfile: "default"})
	model.width = 120
	model.height = 40

	view := model.View()
	for _, fragment := range []string{"GitHub URL", "Saved", "Favorites:", "Recent:"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("input view missing %q", fragment)
		}
	}
}

func TestBrowserViewIncludesRepositoryStatus(t *testing.T) {
	t.Parallel()

	model := newModel("", config.Config{}, RunOptions{ActiveProfile: "work", Token: "ghp_test"})
	model.width = 120
	model.height = 40
	model.mode = modeBrowse
	model.currentRepoURL = "https://github.com/owner/repo"
	model.currentURL = &gh.URL{Owner: "owner", Repo: "repo", Branch: "main", Kind: gh.TargetRepository}
	model.items = []gh.RepoItem{{Name: "main.go", Path: "main.go", ItemType: "file"}}
	model.rateLimit = gh.RateLimitStatus{Authenticated: true, Limit: 5000, Remaining: 4999}
	model.repoMetadata = gh.RepoMetadata{FullName: "owner/repo"}

	view := model.View()
	for _, fragment := range []string{"Repository", "Files (1)", "auth:work 4999/5000"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("browser view missing %q", fragment)
		}
	}
}

func TestOverlayViewsRender(t *testing.T) {
	t.Parallel()

	model := newModel("", config.Config{}, RunOptions{})
	model.width = 120
	model.height = 40
	model.preview = &previewState{
		path:        "main.go",
		content:     "package main\n\n\tfunc main() {}",
		status:      previewReady,
		showNumbers: true,
	}
	model.repoReadme = gh.Readme{Content: "# Title\n\nREADME"}
	model.repoReleases = []gh.Release{{TagName: "v1.0.0", Assets: []gh.ReleaseAsset{{Name: "tool.zip", Size: 10}}}}
	model.releasePicker = &releasePickerState{}

	previewView := model.renderPreview()
	if !strings.Contains(previewView, "Preview") || !strings.Contains(previewView, "[GO]") {
		t.Fatalf("preview view missing expected content")
	}
	if strings.Contains(previewView, "\t") {
		t.Fatalf("preview view should expand tabs before rendering")
	}
	numberedPrefix := mutedTextStyle.Render(fmt.Sprintf("%*d ", 1, 3))
	codeSegment := baseTextStyle.Copy().Foreground(colorFG).Render("    func main() {}")
	if !strings.Contains(previewView, numberedPrefix+codeSegment) {
		t.Fatalf("preview view should style code text with the app background")
	}

	model.preview = nil
	model.infoOverlay = &infoOverlayState{tab: 1}
	infoView := model.renderInfoOverlay()
	if !strings.Contains(infoView, "README") {
		t.Fatalf("info overlay missing README tab")
	}

	model.infoOverlay = nil
	releaseView := model.renderReleasePicker()
	if !strings.Contains(releaseView, "Release Assets") || !strings.Contains(releaseView, "tool.zip") {
		t.Fatalf("release picker missing release asset content")
	}
}

func TestSavePromptRenderIncludesDownloadModes(t *testing.T) {
	t.Parallel()

	model := newModel("", config.Config{}, RunOptions{})
	model.width = 120
	model.height = 40
	model.savePrompt = &savePrompt{
		input:     "/tmp",
		cursor:    4,
		itemCount: 2,
		conflict:  download.ConflictRename,
		output:    download.OutputZip,
	}

	view := model.renderSavePrompt()
	for _, fragment := range []string{"Conflict: rename", "Output:   zip archive", "s/o/r/e", "f/z/t"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("save prompt missing %q", fragment)
		}
	}
}
