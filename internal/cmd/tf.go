package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cleonte/tflings/internal/exercises"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// TfCmd runs a Terraform command of a fix exercise in its sealed environment.
func TfCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:   "tf <exercise name> <terraform arguments...>",
		Short: "Run Terraform inside a fix exercise",
		Long: `Runs the real terraform in the folder and the fake AWS account of a fix
exercise, with a sealed environment: fake credentials, Floci as the endpoint,
nothing from your own shell. This is how you repair a fix exercise.

  tflings tf state-mv1 state list
  tflings tf drift1 plan
  tflings tf drift1 apply -auto-approve

The first use builds the situation (a few seconds). Then run the exercise to check it.`,
		Example: "  tflings tf drift1 apply -refresh-only",
		// Everything after the exercise name belongs to Terraform, flags included.
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}
			if len(args) < 2 {
				return fmt.Errorf("say what Terraform should do, for example: tflings tf %s plan", args[0])
			}
			ex, err := exercises.Find(args[0], infoFile)
			if errors.Is(err, exercises.ErrExerciseNotFound) {
				color.White("No exercise found for '%s'", args[0])
				return err
			}
			if err != nil {
				return err
			}
			code, err := ex.Terraform(cmd.Context(), newEnv(), args[1:], os.Stdin, os.Stdout, os.Stderr)
			if exercises.IsSkipped(err) {
				color.Yellow(err.Error())
				return err
			}
			if err != nil {
				printFailure(ex, err)
				return err
			}
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}
}

// EnvCmd prints the environment of a fix exercise, for a learner who prefers a shell.
func EnvCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:   "env <exercise name>",
		Short: "Print the sealed environment of a fix exercise",
		Long: `Prints the folder and the environment variables that "tflings tf" gives
Terraform. To work in a shell with the same environment:

  cd "$(tflings env drift1 | head -1)" && env -i $(tflings env drift1 | tail -n +2) terraform plan`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ex, err := exercises.Find(args[0], infoFile)
			if err != nil {
				color.White("No exercise found for '%s'", args[0])
				return err
			}
			if ex.Mode != exercises.ModeFix {
				return fmt.Errorf("%s is a %s exercise: only fix exercises have a sealed environment to print", ex.Name, ex.Mode)
			}
			vars, dir, err := ex.SealedEnv(newEnv())
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), dir)
			fmt.Fprintln(cmd.OutOrStdout(), strings.Join(vars, "\n"))
			return nil
		},
	}
}
