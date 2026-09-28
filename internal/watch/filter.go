package watch

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/kyledickey/eye/internal/embeds"
)

// Filter decides which files are worth reloading for and which directories
// are worth watching at all.
type Filter struct {
	Exts   []string   // extensions that trigger a reload, like ".go"
	Ignore []string   // directory names that are never watched, like "vendor"
	Embeds embeds.Set // files compiled into the program, whatever their extension
}

// NewFilter builds a Filter, accepting extensions with or without a leading dot.
func NewFilter(exts, ignore []string) Filter {
	f := Filter{Ignore: ignore}
	for _, ext := range exts {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		f.Exts = append(f.Exts, "."+strings.TrimPrefix(ext, "."))
	}
	return f
}

// Match reports whether a change to the file at path should trigger a reload.
func (f Filter) Match(path string) bool {
	name := filepath.Base(path)

	// Hidden files are almost always editor swap or lock files, and tests
	// never end up in the built binary.
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "_test.go") {
		return false
	}
	return slices.Contains(f.Exts, filepath.Ext(name)) || f.Embeds.Contains(path)
}

// Skip reports whether the directory at path should be left unwatched.
func (f Filter) Skip(path string) bool {
	name := filepath.Base(path)
	if name == "." || name == ".." {
		return false
	}
	return strings.HasPrefix(name, ".") || slices.Contains(f.Ignore, name)
}
