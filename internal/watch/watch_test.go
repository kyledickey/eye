package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWatcher writes files into a fresh directory tree and checks whether the
// watcher reports them as a change.
func TestWatcher(t *testing.T) {
	tests := []struct {
		name  string
		setup []string // directories that exist before watching starts
		mkdir string   // directory created after watching starts, if any
		write string   // file written after watching starts
		want  bool     // whether a change should be reported
	}{
		{name: "go file in root", write: "main.go", want: true},
		{name: "go file in subdir", setup: []string{"app"}, write: "app/app.go", want: true},
		{name: "go file in new subdir", mkdir: "fresh", write: "fresh/fresh.go", want: true},
		{name: "unwatched extension", write: "notes.txt", want: false},
		{name: "test file", write: "main_test.go", want: false},
		{name: "ignored dir", setup: []string{"vendor"}, write: "vendor/lib.go", want: false},
		{name: "hidden dir", setup: []string{".cache"}, write: ".cache/gen.go", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, dir := range tt.setup {
				mkdir(t, filepath.Join(root, dir))
			}

			filter := NewFilter([]string{"go"}, []string{"vendor"})
			w, err := New([]string{root}, filter, 10*time.Millisecond)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			changes := make(chan []string)
			go w.Run(ctx, changes)

			if tt.mkdir != "" {
				mkdir(t, filepath.Join(root, tt.mkdir))
				// Give the watcher a beat to pick up the new directory.
				time.Sleep(50 * time.Millisecond)
			}

			path := filepath.Join(root, tt.write)
			if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}

			select {
			case files := <-changes:
				if !tt.want {
					t.Fatalf("got unexpected change %q", files)
				}
				if len(files) != 1 || files[0] != path {
					t.Errorf("files = %q, want [%q]", files, path)
				}
			case <-time.After(300 * time.Millisecond):
				if tt.want {
					t.Fatal("timed out waiting for change")
				}
			}
		})
	}
}

// TestWatcherBatches checks that a burst of writes is reported as one batch.
func TestWatcherBatches(t *testing.T) {
	root := t.TempDir()

	w, err := New([]string{root}, NewFilter([]string{"go"}, nil), 50*time.Millisecond)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	changes := make(chan []string)
	go w.Run(t.Context(), changes)

	for _, name := range []string{"a.go", "b.go", "c.go", "a.go"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	select {
	case files := <-changes:
		if len(files) != 3 {
			t.Errorf("got %d files %q, want 3", len(files), files)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for change")
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}
