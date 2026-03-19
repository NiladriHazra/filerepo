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

func TestMatchesSearch(t *testing.T) {
	t.Parallel()

	size := uint64(10)
	itemFile := gh.RepoItem{Name: "main.go", Path: "cmd/app/main.go", ItemType: "file", Size: &size}
	itemDir := gh.RepoItem{Name: "internal", Path: "internal", ItemType: "dir"}
	itemLFS := gh.RepoItem{Name: "model.bin", Path: "assets/model.bin", ItemType: "file", Size: &size, LFSOID: "abc"}

	tests := []struct {
		name  string
		query string
		item  gh.RepoItem
		want  bool
	}{
		{name: "substring match", query: "main", item: itemFile, want: true},
		{name: "fuzzy match", query: "cmdmg", item: itemFile, want: true},
		{name: "ext filter matches", query: "ext:go", item: itemFile, want: true},
		{name: "ext filter rejects", query: "ext:rs", item: itemFile, want: false},
		{name: "type dir matches", query: "type:dir", item: itemDir, want: true},
		{name: "type file rejects dir", query: "type:file", item: itemDir, want: false},
		{name: "type lfs matches", query: "type:lfs", item: itemLFS, want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := matchesSearch(tt.item, parseSearchQuery(tt.query)); got != tt.want {
				t.Fatalf("matchesSearch(%q) = %t want %t", tt.query, got, tt.want)
			}
		})
	}
}
