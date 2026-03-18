package ui

import (
	"testing"

	gh "github.com/NiladriHazra/filerepo/internal/github"
)

func TestRepoItemsForPath(t *testing.T) {
	t.Parallel()

	size := uint64(10)
	items := []gh.RepoItem{
		{Name: "lib.rs", Path: "src/lib.rs", ItemType: "file", Size: &size},
		{Name: "ui", Path: "src/ui", ItemType: "dir"},
		{Name: "main.rs", Path: "src/main.rs", ItemType: "file", Size: &size},
		{Name: "README.md", Path: "README.md", ItemType: "file", Size: &size},
	}

	root := repoItemsForPath(items, "")
	if len(root) != 1 || root[0].Path != "README.md" {
		t.Fatalf("unexpected root items: %+v", root)
	}

	src := repoItemsForPath(items, "src")
	if got, want := len(src), 3; got != want {
		t.Fatalf("unexpected src child count: got %d want %d", got, want)
	}
}
