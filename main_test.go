package main

import (
	"slices"
	"testing"
)

// TestSplitArgs runs a command line through eye's real flag parsing and checks
// which package gets built and which arguments are handed to the program.
func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name     string
		argv     []string
		wantPkg  string
		wantArgs []string
	}{
		{name: "nothing defaults to current dir", argv: nil, wantPkg: "."},
		{name: "package only", argv: []string{"./cmd/app"}, wantPkg: "./cmd/app"},
		{name: "program flags after package", argv: []string{"./cmd/app", "-addr", ":3000"}, wantPkg: "./cmd/app", wantArgs: []string{"-addr", ":3000"}},
		{name: "program args after package", argv: []string{"./cmd/app", "serve", "-v"}, wantPkg: "./cmd/app", wantArgs: []string{"serve", "-v"}},
		{name: "eye flags before package", argv: []string{"--debug", "-w", "..", "./cmd/app", "-x"}, wantPkg: "./cmd/app", wantArgs: []string{"-x"}},
		{name: "eye flag names after package belong to the program", argv: []string{"./cmd/app", "--debug"}, wantPkg: "./cmd/app", wantArgs: []string{"--debug"}},
		{name: "dash after package is dropped", argv: []string{"./cmd/app", "--", "-addr", ":3000"}, wantPkg: "./cmd/app", wantArgs: []string{"-addr", ":3000"}},
		{name: "leading dash means current dir", argv: []string{"--", "-addr", ":3000"}, wantPkg: ".", wantArgs: []string{"-addr", ":3000"}},
		{name: "only a dash", argv: []string{"--"}, wantPkg: "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := command()
			if err := cmd.ParseFlags(tt.argv); err != nil {
				t.Fatalf("ParseFlags: %v", err)
			}

			pkg, args := splitArgs(cmd.Flags().Args(), cmd.ArgsLenAtDash())
			if pkg != tt.wantPkg {
				t.Errorf("pkg = %q, want %q", pkg, tt.wantPkg)
			}
			if !slices.Equal(args, tt.wantArgs) {
				t.Errorf("args = %q, want %q", args, tt.wantArgs)
			}
		})
	}
}
