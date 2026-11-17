package cmd

import (
	"errors"
	"io/fs"

	"github.com/cleonte/tflings/internal/exercises"
	"github.com/cleonte/tflings/internal/workspace"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ResetCmd restores an exercise to its original state.
func ResetCmd(src fs.FS, root string) *cobra.Command {
	return &cobra.Command{
		Use:   "reset <exercise name>",
		Short: "Restore an exercise to its original, broken state",
		Long: `Overwrites the exercise file with the original from the binary, which also
brings back the 'I AM NOT DONE' line, and forgets that you had solved it. Your
changes to that file are lost. An exercise that uses Floci gets a fresh, empty
AWS account.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := workspace.Reset(src, root, args[0])
			if errors.Is(err, exercises.ErrExerciseNotFound) {
				color.White("No exercise found for '%s'", args[0])
				return reported(err)
			}
			if err != nil {
				return err
			}
			if err := (&exercises.Env{}).ForgetRun(args[0]); err != nil {
				color.Yellow("Restored %s, but could not delete its saved state: %s", p, err)
			}
			if err := exercises.RecordReset(root, args[0]); err != nil {
				color.Yellow("Restored %s, but could not save the reset in %s: %s", p, exercises.ProgressFile, err)
				return reported(err)
			}
			color.Green("Restored %s", p)
			return nil
		},
	}
}
