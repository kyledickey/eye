// Package watch reports settled changes to files under a set of directories.
package watch

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/kyledickey/eye/internal/embeds"
)

// Watcher recursively watches directories and groups rapid bursts of file
// changes into a single batch once things have been quiet for a moment.
type Watcher struct {
	fs    *fsnotify.Watcher
	delay time.Duration

	mu     sync.Mutex // guards filter, whose embeds change between builds
	filter Filter
}

// New starts watching every directory under roots that the filter allows.
func New(roots []string, filter Filter, delay time.Duration) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{fs: fsw, filter: filter, delay: delay}
	for _, root := range roots {
		if err := w.add(root); err != nil {
			fsw.Close()
			return nil, err
		}
	}
	return w, nil
}

// Dirs returns the directories currently being watched.
func (w *Watcher) Dirs() []string {
	return w.fs.WatchList()
}

// SetEmbeds replaces the embedded files that trigger a reload.
func (w *Watcher) SetEmbeds(s embeds.Set) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.filter.Embeds = s
}

// currentFilter returns a snapshot of the filter that is safe to use freely.
func (w *Watcher) currentFilter() Filter {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.filter
}

// Run sends each settled batch of changed files to changes until ctx is done.
// It closes the underlying watcher before returning.
func (w *Watcher) Run(ctx context.Context, changes chan<- []string) error {
	defer w.fs.Close()

	// The timer only exists while changes are pending, so a nil channel
	// keeps its select case asleep the rest of the time.
	var (
		pending = map[string]bool{}
		timer   *time.Timer
		settled <-chan time.Time
	)

	for {
		select {
		case <-ctx.Done():
			return nil

		case err, ok := <-w.fs.Errors:
			if !ok {
				return nil
			}
			slog.Warn("watch error", "err", err)

		case ev, ok := <-w.fs.Events:
			if !ok {
				return nil
			}
			if !w.handle(ev) {
				continue
			}

			pending[ev.Name] = true
			if timer == nil {
				timer = time.NewTimer(w.delay)
			} else {
				timer.Reset(w.delay)
			}
			settled = timer.C

		case <-settled:
			files := make([]string, 0, len(pending))
			for name := range pending {
				files = append(files, name)
			}
			slices.Sort(files)
			slog.Debug("changes settled", "files", len(files))

			pending, settled = map[string]bool{}, nil
			select {
			case changes <- files:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// handle reacts to a single event and reports whether it counts as a change.
func (w *Watcher) handle(ev fsnotify.Event) bool {
	if ev.Has(fsnotify.Chmod) && !ev.Has(fsnotify.Write) {
		return false
	}

	// New directories need watching too, or nothing inside them is seen.
	if ev.Has(fsnotify.Create) {
		if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
			if err := w.add(ev.Name); err != nil {
				slog.Warn("could not watch new directory", "dir", ev.Name, "err", err)
			}
			return false
		}
	}

	if !w.currentFilter().Match(ev.Name) {
		slog.Debug("ignored", "op", ev.Op, "file", ev.Name)
		return false
	}
	slog.Debug("changed", "op", ev.Op, "file", ev.Name)
	return true
}

// add watches root and every directory beneath it that the filter allows.
func (w *Watcher) add(root string) error {
	start := time.Now()
	count := 0
	filter := w.currentFilter()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// A directory can vanish mid-walk, that's fine.
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if filter.Skip(path) {
			slog.Debug("skipping", "dir", path)
			return filepath.SkipDir
		}
		count++
		return w.fs.Add(path)
	})

	slog.Debug("watching", "root", root, "dirs", count, "took", time.Since(start))
	return err
}
