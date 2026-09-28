package watch

import (
	"slices"
	"testing"
)

// TestNewFilter checks that extensions are normalized to a single leading dot
// and that blank entries are dropped.
func TestNewFilter(t *testing.T) {
	tests := []struct {
		name string
		exts []string
		want []string
	}{
		{"bare names get a dot", []string{"go", "mod"}, []string{".go", ".mod"}},
		{"dotted names are kept", []string{".go", ".sum"}, []string{".go", ".sum"}},
		{"blanks are dropped", []string{"go", "", "  "}, []string{".go"}},
		{"whitespace is trimmed", []string{" go "}, []string{".go"}},
		{"nothing in, nothing out", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFilter(tt.exts, nil).Exts
			if !slices.Equal(got, tt.want) {
				t.Errorf("Exts = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFilterMatch checks which changed files are allowed to trigger a reload.
func TestFilterMatch(t *testing.T) {
	f := NewFilter([]string{"go", "mod"}, nil)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"go source", "main.go", true},
		{"nested go source", "internal/app/app.go", true},
		{"go.mod", "go.mod", true},
		{"unwatched extension", "README.md", false},
		{"no extension", "Makefile", false},
		{"test file", "main_test.go", false},
		{"hidden file", ".main.go", false},
		{"emacs lock file", "internal/.#app.go", false},
		{"vim backup", "main.go~", false},
		{"vim swap", ".main.go.swp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.Match(tt.path); got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestFilterSkip checks which directories are left out of the watch.
func TestFilterSkip(t *testing.T) {
	f := NewFilter(nil, []string{"vendor", "node_modules"})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"current dir", ".", false},
		{"parent dir", "..", false},
		{"plain dir", "internal", false},
		{"nested plain dir", "internal/watch", false},
		{"ignored dir", "vendor", true},
		{"nested ignored dir", "web/node_modules", true},
		{"hidden dir", ".git", true},
		{"nested hidden dir", "tools/.cache", true},
		{"similar name is not ignored", "vendored", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.Skip(tt.path); got != tt.want {
				t.Errorf("Skip(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
