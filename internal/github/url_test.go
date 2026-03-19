package github

import "testing"

func TestParseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    URL
		wantErr bool
	}{
		{
			name: "root repository",
			raw:  "https://github.com/rust-lang/rust",
			want: URL{Owner: "rust-lang", Repo: "rust", Branch: "main", Kind: TargetRepository},
		},
		{
			name: "tree path",
			raw:  "https://github.com/rust-lang/rust/tree/master/src/tools",
			want: URL{Owner: "rust-lang", Repo: "rust", Branch: "master", Path: "src/tools", Kind: TargetRepository},
		},
		{
			name: "blob path",
			raw:  "https://github.com/owner/repo/blob/main/src/lib.rs",
			want: URL{Owner: "owner", Repo: "repo", Branch: "main", Path: "src/lib.rs", Kind: TargetRepository},
		},
		{
			name: "commit url",
			raw:  "https://github.com/owner/repo/commit/abc123def456",
			want: URL{Owner: "owner", Repo: "repo", Branch: "abc123def456", Kind: TargetRepository},
		},
		{
			name: "compare url",
			raw:  "https://github.com/owner/repo/compare/main...feature",
			want: URL{
				Owner:       "owner",
				Repo:        "repo",
				Branch:      "feature",
				Kind:        TargetCompare,
				CompareBase: "main",
				CompareHead: "feature",
			},
		},
		{
			name: "pull request url",
			raw:  "https://github.com/owner/repo/pull/42",
			want: URL{
				Owner:      "owner",
				Repo:       "repo",
				Branch:     "main",
				Kind:       TargetPullRequest,
				PullNumber: 42,
			},
		},
		{
			name:    "non github host",
			raw:     "https://gitlab.com/user/repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseURL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseURL returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("ParseURL mismatch: got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestURLWebURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  URL
		want string
	}{
		{
			name: "repo root",
			url:  URL{Owner: "owner", Repo: "repo", Branch: "main"},
			want: "https://github.com/owner/repo",
		},
		{
			name: "tree path",
			url:  URL{Owner: "owner", Repo: "repo", Branch: "dev", Path: "src"},
			want: "https://github.com/owner/repo/tree/dev/src",
		},
		{
			name: "compare",
			url: URL{
				Owner:       "owner",
				Repo:        "repo",
				Kind:        TargetCompare,
				CompareBase: "main",
				CompareHead: "feature",
			},
			want: "https://github.com/owner/repo/compare/main...feature",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.url.WebURL(); got != tt.want {
				t.Fatalf("WebURL mismatch: got %q want %q", got, tt.want)
			}
		})
	}
}
