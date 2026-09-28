// Package embeds finds the files a Go program compiles in with go:embed.
package embeds

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"time"
)

// Set holds the embedded files of a program, and the directories they were
// embedded from, all as absolute paths.
type Set struct {
	files map[string]bool
	dirs  map[string]bool
}

// pkg is the part of `go list` output we care about.
type pkg struct {
	Dir        string   // absolute path to the package
	EmbedFiles []string // embedded files, relative to Dir
}

// List asks the go tool which files pkg and its dependencies embed.
func List(ctx context.Context, target string) (Set, error) {
	start := time.Now()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-json=Dir,EmbedFiles", target)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Set{}, fmt.Errorf("go list: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}

	// go list prints one JSON object per package, back to back.
	var pkgs []pkg
	dec := json.NewDecoder(&stdout)
	for {
		var p pkg
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return Set{}, err
		}
		pkgs = append(pkgs, p)
	}

	s := newSet(pkgs)
	slog.Debug("found embeds", "files", len(s.files), "took", time.Since(start).Round(time.Millisecond))
	return s, nil
}

// newSet collects every embedded and relevant file.
func newSet(pkgs []pkg) Set {
	s := Set{files: map[string]bool{}, dirs: map[string]bool{}}
	for _, p := range pkgs {
		for _, rel := range p.EmbedFiles {
			s.files[filepath.Join(p.Dir, rel)] = true
			for dir := filepath.Dir(rel); dir != "."; dir = filepath.Dir(dir) {
				s.dirs[filepath.Join(p.Dir, dir)] = true
			}
		}
	}
	return s
}

// Contains reports whether path is embedded, or in an embedded directory.
func (s Set) Contains(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return s.files[abs] || s.dirs[filepath.Dir(abs)]
}

func (s Set) Len() int {
	return len(s.files)
}
