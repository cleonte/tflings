package cmd

import (
	"errors"
	"fmt"

	"github.com/cleonte/tflings/internal/exercises"
	"github.com/fatih/color"
)

// printFailure explains why an exercise is not solved yet.
func printFailure(ex exercises.Exercise, err error) {
	var failed *exercises.FailedError
	if !errors.As(err, &failed) {
		color.Red("❌ %s: %s", ex.Name, err)
		return
	}
	color.Red("❌ %s is not solved yet:\n", ex.Name)
	for _, line := range failed.Lines {
		color.Red("  • %s", line)
	}
	if ex.MarkerGone() && ex.State() == exercises.Pending {
		color.Yellow("\nThe 'I AM NOT DONE' line is gone, but %s has not passed yet, so it still counts as pending.", ex.Name)
	}
	if len(failed.Details) > 0 {
		fmt.Println()
		for _, line := range failed.Details {
			fmt.Println(line)
		}
	}
	if ex.Mode == exercises.ModeFix {
		color.White("\nThe ticket is in %s", ex.File())
	} else {
		color.White("\nEdit %s", ex.Location())
	}
	color.White("Need a nudge? tflings hint %s", ex.Name)
}
