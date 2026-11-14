package engine

import (
	"slices"
	"testing"
)

func TestFlagsPerCommand(t *testing.T) {
	tf := &terraform{}
	cases := map[string][]string{
		"plan":     {"-no-color", "-input=false"},
		"init":     {"-no-color", "-input=false"},
		"validate": {"-no-color"},
		"show":     {"-no-color"},
		"version":  nil,
	}
	for cmd, want := range cases {
		if got := tf.Flags(cmd); !slices.Equal(got, want) {
			t.Errorf("Flags(%q) = %v, want %v", cmd, got, want)
		}
	}
}
