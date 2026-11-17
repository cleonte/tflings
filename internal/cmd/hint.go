package cmd

import (
	"errors"

	"github.com/cleonte/tflings/internal/exercises"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// HintCmd shows the hint and the documentation link of an exercise.
func HintCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:           "hint <exercise name>",
		Short:         "Show the hint of an exercise",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			ex, err := exercises.Find(args[0], infoFile)
			if errors.Is(err, exercises.ErrExerciseNotFound) {
				color.White("No exercise found for '%s'", args[0])
				return reported(err)
			}
			if err != nil {
				return err
			}
			color.Cyan(ex.Hint)
			if ex.Docs != "" {
				color.White("\nDocs: %s", ex.Docs)
			}
			return nil
		},
	}
}
