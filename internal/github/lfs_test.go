package github

import "testing"

func TestParseLFSPointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want *LFSPointer
	}{
		{
			name: "valid pointer",
			text: "version https://git-lfs.github.com/spec/v1\noid sha256:abc123\nsize 42",
			want: &LFSPointer{OID: "abc123", Size: 42},
		},
		{
			name: "regular file",
			text: "hello world",
			want: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseLFSPointer(tt.text)
			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("expected nil pointer, got %+v", got)
			case tt.want != nil && got == nil:
				t.Fatalf("expected pointer %+v, got nil", tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Fatalf("pointer mismatch: got %+v want %+v", got, tt.want)
			}
		})
	}
}
