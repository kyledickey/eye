package main

import (
	"bytes"
	"testing"
)

// TestPrefixed checks that each write gets the prefix once, and that the
// reported length is the caller's, not including the prefix.
func TestPrefixed(t *testing.T) {
	tests := []struct {
		name   string
		writes []string
		want   string
	}{
		{"one line", []string{"started\n"}, "eye ◉ started\n"},
		{"two lines", []string{"changed\n", "reloaded\n"}, "eye ◉ changed\neye ◉ reloaded\n"},
		{"empty write", []string{""}, "eye ◉ "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := prefixed{w: &buf, prefix: []byte(mark)}

			for _, s := range tt.writes {
				n, err := p.Write([]byte(s))
				if err != nil {
					t.Fatalf("Write: %v", err)
				}
				if n != len(s) {
					t.Errorf("Write returned %d, want %d", n, len(s))
				}
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}
