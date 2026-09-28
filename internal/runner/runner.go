// Package runner builds a Go package and keeps a single copy of it running.
package runner

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"
)

// stopTimeout is how long a program gets to shut down before it is killed.
const stopTimeout = 5 * time.Second

// BuildError holds the compiler output from a failed build.
type BuildError struct {
	Output string
}

func (e *BuildError) Error() string {
	return "build failed"
}

// Runner builds Pkg into a temporary binary and runs it with Args.
type Runner struct {
	Pkg  string
	Args []string

	dir  string
	bin  string
	proc *process
}

// process is one running copy of the program.
type process struct {
	cmd     *exec.Cmd
	done    chan struct{} // closed once the process has exited
	stopped atomic.Bool   // set when we asked it to exit
}

// New creates a Runner with its own scratch directory for binaries.
func New(pkg string, args []string) (*Runner, error) {
	dir, err := os.MkdirTemp("", "eye-*")
	if err != nil {
		return nil, err
	}
	return &Runner{
		Pkg:  pkg,
		Args: args,
		dir:  dir,
		bin:  filepath.Join(dir, "app"),
	}, nil
}

// Build compiles the package, returning a *BuildError if the compiler complains.
func (r *Runner) Build(ctx context.Context) error {
	start := time.Now()

	var out bytes.Buffer
	// VCS stamping only slows down dev builds, and fails outright in odd repos.
	cmd := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-o", r.bin, r.Pkg)
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	slog.Debug("build finished", "pkg", r.Pkg, "took", time.Since(start).Round(time.Millisecond))

	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return &BuildError{Output: out.String()}
	}
	return err
}

// Start launches the last successful build with its output sent to the
// terminal.
func (r *Runner) Start() error {
	cmd := exec.Command(r.bin, r.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	isolate(cmd)

	if err := cmd.Start(); err != nil {
		return err
	}
	slog.Debug("process started", "pid", cmd.Process.Pid)

	p := &process{cmd: cmd, done: make(chan struct{})}
	r.proc = p

	go func() {
		defer close(p.done)
		cmd.Wait()

		// Only speak up if the program ended on its own, not because we stopped it.
		if p.stopped.Load() {
			return
		}
		if code := cmd.ProcessState.ExitCode(); code != 0 {
			slog.Warn("exited", "code", code)
		} else {
			slog.Info("exited")
		}
	}()
	return nil
}

// Stop asks the running program to exit, killing it if it takes too long.
// It is a no-op when nothing is running.
func (r *Runner) Stop() {
	p := r.proc
	if p == nil {
		return
	}
	r.proc = nil

	p.stopped.Store(true)
	select {
	case <-p.done:
		return
	default:
	}

	start := time.Now()
	if err := interrupt(p.cmd); err != nil {
		slog.Debug("interrupt failed", "err", err)
	}

	select {
	case <-p.done:
		slog.Debug("process stopped", "took", time.Since(start))
	case <-time.After(stopTimeout):
		slog.Warn("program did not stop in time, killing it", "after", stopTimeout)
		kill(p.cmd)
		<-p.done
	}
}

// Close stops the program and removes the scratch directory.
func (r *Runner) Close() {
	r.Stop()
	if err := os.RemoveAll(r.dir); err != nil {
		slog.Debug("cleanup failed", "dir", r.dir, "err", err)
	}
}
