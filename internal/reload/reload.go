// Package reload ties the watcher and runner together: build, run, and
// rebuild whenever something changes.
package reload

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kyledickey/eye/internal/embeds"
	"github.com/kyledickey/eye/internal/runner"
	"github.com/kyledickey/eye/internal/watch"
)

// Config describes what to run and what to watch.
type Config struct {
	Pkg    string        // package to build, like "." or "./cmd/app"
	Args   []string      // arguments passed to the program
	Watch  []string      // directories to watch
	Exts   []string      // file extensions that trigger a reload
	Ignore []string      // directory names to skip
	Delay  time.Duration // how long changes must settle before reloading
}

// Run builds and runs the program, reloading it on every change until ctx
// is done.
func Run(ctx context.Context, cfg Config) error {
	filter := watch.NewFilter(cfg.Exts, cfg.Ignore)
	w, err := watch.New(cfg.Watch, filter, cfg.Delay)
	if err != nil {
		return fmt.Errorf("watch: %w", err)
	}

	r, err := runner.New(cfg.Pkg, cfg.Args)
	if err != nil {
		return fmt.Errorf("runner: %w", err)
	}
	defer r.Close()

	changes := make(chan []string)
	watchErr := make(chan error, 1)
	go func() { watchErr <- w.Run(ctx, changes) }()

	attrs := []any{"pkg", cfg.Pkg, "dirs", len(w.Dirs())}
	if n := refreshEmbeds(ctx, w, cfg.Pkg); n > 0 {
		attrs = append(attrs, "embeds", n)
	}
	slog.Info("watching", attrs...)
	restart(ctx, r, "started")

	for {
		select {
		case <-ctx.Done():
			slog.Debug("shutting down")
			return nil

		case err := <-watchErr:
			return err

		case files := <-changes:
			attrs := []any{"file", files[0]}
			if len(files) > 1 {
				attrs = append(attrs, "more", len(files)-1)
			}
			slog.Info("changed", attrs...)
			restart(ctx, r, "reloaded")

			// The change may have added or removed a go:embed.
			refreshEmbeds(ctx, w, cfg.Pkg)
		}
	}
}

// refreshEmbeds tells the watcher which files the program embeds.
func refreshEmbeds(ctx context.Context, w *watch.Watcher, pkg string) int {
	set, err := embeds.List(ctx, pkg)
	if err != nil {
		slog.Debug("could not list embeds", "err", err)
		return 0
	}
	w.SetEmbeds(set)
	return set.Len()
}

// restart stops the program, rebuilds it, and starts it again.
func restart(ctx context.Context, r *runner.Runner, done string) {
	start := time.Now()
	r.Stop()

	err := r.Build(ctx)
	if ctx.Err() != nil {
		return
	}

	var buildErr *runner.BuildError
	switch {
	case errors.As(err, &buildErr):
		slog.Error("build failed")
		fmt.Fprint(os.Stderr, buildErr.Output)
		return
	case err != nil:
		slog.Error("build failed", "err", err)
		return
	}

	if err := r.Start(); err != nil {
		slog.Error("could not start", "err", err)
		return
	}
	slog.Info(done, "took", time.Since(start).Round(time.Millisecond))
}
