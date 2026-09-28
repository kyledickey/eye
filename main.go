// Command eye rebuilds and reruns a Go program whenever its files change.
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kyledickey/eye/internal/reload"
)

func main() {
	setupLogger(false)
	if err := command().Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func command() *cobra.Command {
	var (
		cfg   reload.Config
		debug bool
	)

	cmd := &cobra.Command{
		Use:   "eye [flags] [package] [args...]",
		Short: "Rebuild and rerun a Go program whenever its files change",
		Long: `Rebuild and rerun a Go program whenever its files change.

Flags for eye go before the package. Everything after the package is passed
to the program untouched. To pass arguments to the package in the current
directory, use "eye . -addr :3000" or "eye -- -addr :3000".`,
		Example: `  eye
  eye ./cmd/server
  eye ./cmd/server -addr :3000
  eye --debug -w . -w ../shared ./cmd/server`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,

		PreRun: func(cmd *cobra.Command, args []string) {
			setupLogger(debug)
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Pkg, cfg.Args = splitArgs(args, cmd.ArgsLenAtDash())
			slog.Debug("config", "pkg", cfg.Pkg, "args", cfg.Args, "watch", cfg.Watch,
				"ext", cfg.Exts, "ignore", cfg.Ignore, "delay", cfg.Delay)

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return reload.Run(ctx, cfg)
		},
	}

	flags := cmd.Flags()

	flags.SetInterspersed(false)
	flags.StringSliceVarP(&cfg.Watch, "watch", "w", []string{"."}, "directories to watch")
	flags.StringSliceVarP(&cfg.Exts, "ext", "e", []string{"go", "mod", "sum"}, "file extensions that trigger a reload")
	flags.StringSliceVarP(&cfg.Ignore, "ignore", "i", []string{"vendor", "node_modules", "testdata", "tmp"}, "directory names to skip (hidden directories are always skipped)")
	flags.DurationVarP(&cfg.Delay, "delay", "d", 100*time.Millisecond, "how long changes must settle before reloading")
	flags.BoolVar(&debug, "debug", false, "show debug logs")

	return cmd
}

func splitArgs(args []string, dash int) (pkg string, rest []string) {
	// A leading "--" means no package was given, only program arguments.
	if dash == 0 || len(args) == 0 {
		return ".", args
	}

	rest = args[1:]
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}
	return args[0], rest
}
