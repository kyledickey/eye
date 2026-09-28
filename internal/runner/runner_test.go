package runner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBuild compiles small throwaway programs and checks that failures come
// back as a BuildError carrying the compiler's complaint.
func TestBuild(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr string // substring of the compiler output, empty for success
	}{
		{
			name:   "valid program",
			source: "package main\n\nfunc main() {}\n",
		},
		{
			name:    "syntax error",
			source:  "package main\n\nfunc main() {\n",
			wantErr: "syntax error",
		},
		{
			name:    "type error",
			source:  "package main\n\nfunc main() { var x int = \"no\" }\n",
			wantErr: "cannot use",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRunner(t, tt.source)

			err := r.Build(t.Context())
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Build: %v", err)
				}
				return
			}

			var buildErr *BuildError
			if !errors.As(err, &buildErr) {
				t.Fatalf("Build error = %v, want *BuildError", err)
			}
			if !strings.Contains(buildErr.Output, tt.wantErr) {
				t.Errorf("output = %q, want it to contain %q", buildErr.Output, tt.wantErr)
			}
		})
	}
}

// TestStop checks that Stop ends a running program with an interrupt alone,
// whether the program dies on the spot or takes a moment to clean up.
func TestStop(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "exits on interrupt",
			source: `package main

import "time"

func main() { time.Sleep(time.Hour) }
`,
		},
		{
			name: "exits after cleanup",
			source: `package main

import (
	"os"
	"os/signal"
	"time"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	time.Sleep(50 * time.Millisecond)
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRunner(t, tt.source)
			if err := r.Build(t.Context()); err != nil {
				t.Fatalf("Build: %v", err)
			}
			if err := r.Start(); err != nil {
				t.Fatalf("Start: %v", err)
			}
			p := r.proc

			start := time.Now()
			r.Stop()

			select {
			case <-p.done:
			default:
				t.Fatal("process still running after Stop")
			}
			if took := time.Since(start); took >= stopTimeout {
				t.Errorf("Stop took %v, should not have needed a kill", took)
			}
		})
	}
}

// TestStopIdle checks that stopping with nothing running is harmless.
func TestStopIdle(t *testing.T) {
	r, err := New(".", nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r.Stop()
	r.Close()

	if _, err := os.Stat(r.dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("scratch dir still exists after Close")
	}
}

// newRunner writes source into a fresh module, moves into it, and returns a
// Runner for it that is cleaned up when the test ends.
func newRunner(t *testing.T, source string) *Runner {
	t.Helper()

	dir := t.TempDir()
	write(t, filepath.Join(dir, "go.mod"), "module example\n\ngo 1.25\n")
	write(t, filepath.Join(dir, "main.go"), source)
	t.Chdir(dir)

	r, err := New(".", nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(r.Close)
	return r
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
