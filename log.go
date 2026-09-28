package main

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

const (
	mark      = "eye ◉ "
	markColor = "\033[95meye ◉\033[0m "
)

func setupLogger(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	info, err := os.Stderr.Stat()
	color := err == nil && info.Mode()&os.ModeCharDevice != 0

	prefix := mark
	if color {
		prefix = markColor
	}

	out := prefixed{w: os.Stderr, prefix: []byte(prefix)}
	slog.SetDefault(slog.New(tint.NewTextHandler(out, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
		NoColor:    !color,
	})))
}

type prefixed struct {
	w      io.Writer
	prefix []byte
}

func (p prefixed) Write(b []byte) (int, error) {
	// One write for prefix and line, so concurrent output can't split them.
	line := make([]byte, 0, len(p.prefix)+len(b))
	line = append(append(line, p.prefix...), b...)
	if _, err := p.w.Write(line); err != nil {
		return 0, err
	}
	return len(b), nil
}
