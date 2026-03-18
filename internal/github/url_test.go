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
			want: URL{Owner: "rust-lang", Repo: "rust", Branch: "main"},
		},
		{
			name: "tree path",
			raw:  "https://github.com/rust-lang/rust/tree/master/src/tools",
			want: URL{Owner: "rust-lang", Repo: "rust", Branch: "master", Path: "src/tools"},
		},
		{
			name: "blob path",
			raw:  "https://github.com/owner/repo/blob/main/src/lib.rs",
			want: URL{Owner: "owner", Repo: "repo", Branch: "main", Path: "src/lib.rs"},
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
