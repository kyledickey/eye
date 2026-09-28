package embeds

import (
	"os"
	"path/filepath"
	"testing"
)

// TestContains checks which paths count as embedded, given a web package that
// embeds a whole directory and a db package that embeds loose files.
func TestContains(t *testing.T) {
	s := newSet([]pkg{
		{Dir: "/app/web", EmbedFiles: []string{"site/index.html", "site/fonts/mono.woff2"}},
		{Dir: "/app/db", EmbedFiles: []string{"schema.sql"}},
		{Dir: "/app/cmd"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"embedded file", "/app/web/site/index.html", true},
		{"embedded file in subdir", "/app/web/site/fonts/mono.woff2", true},
		{"new file in embedded dir", "/app/web/site/style.css", true},
		{"new file in embedded subdir", "/app/web/site/fonts/sans.woff2", true},
		{"loose embedded file", "/app/db/schema.sql", true},
		{"neighbor of loose embedded file", "/app/db/notes.md", false},
		{"neighbor of embedded dir", "/app/web/README.md", false},
		{"package without embeds", "/app/cmd/main.txt", false},
		{"unrelated path", "/elsewhere/site/index.html", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Contains(tt.path); got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestList runs the real go tool against a small module and checks that
// embedded files are found, including those of dependencies.
func TestList(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example\n\ngo 1.25\n")
	write(t, dir, "main.go", "package main\n\nimport _ \"example/web\"\n\nfunc main() {}\n")
	write(t, dir, "web/web.go", "package web\n\nimport \"embed\"\n\n//go:embed site\nvar Site embed.FS\n")
	write(t, dir, "web/site/index.html", "<h1>hi</h1>\n")
	write(t, dir, "web/site/style.css", "h1 {}\n")
	t.Chdir(dir)

	s, err := List(t.Context(), ".")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"embedded html", "web/site/index.html", true},
		{"embedded css", "web/site/style.css", true},
		{"go source", "web/web.go", false},
	}

	if s.Len() != 2 {
		t.Errorf("Len = %d, want 2", s.Len())
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Contains(tt.path); got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
