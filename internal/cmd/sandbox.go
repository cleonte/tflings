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

// SandboxCmd is a scratch folder for trying Terraform outside the exercises.
func SandboxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sandbox [terraform arguments... | --reset]",
		Short: "Try Terraform freely in a scratch folder with its own fake AWS account",
		Long: `The sandbox is a folder of your own, with the same sealed environment the
exercises use: fake credentials, the local Floci as the only endpoint, nothing
from your shell. Write Terraform files in it and run any Terraform command there:

  tflings sandbox                 print the folder and the environment, and make the folder
  tflings sandbox init
  tflings sandbox apply -auto-approve
  tflings sandbox --reset         delete its files and state, and start in a new empty account

Floci has to be running for AWS resources (tflings aws up), and the AWS provider
comes from tflings's cache (tflings doctor --prefetch), so use the version in
the exercises (` + "`~> 6.0`" + `). Nothing here talks to a real AWS.`,
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			env := newEnv()
			switch {
			case len(args) == 1 && args[0] == "--reset":
				if err := env.SandboxReset(); err != nil {
					return err
				}
				color.Green("The sandbox is empty again, in a new fake AWS account.")
				color.White("Resources the old account built stay in Floci until `tflings aws reset`.")
				return nil
			case len(args) == 0:
				folder, vars, err := env.SandboxEnv()
				if err != nil {
					return err
				}
				fmt.Println(folder)
				fmt.Println(strings.Join(vars, "\n"))
				return nil
			}
			if env.FlociUp != nil && !env.FlociUp(cmd.Context()) {
				color.Yellow("Floci is not running: AWS resources need `tflings aws up`.")
			}
			code, err := env.Sandbox(cmd.Context(), args, os.Stdin, os.Stdout, os.Stderr)
			if exercises.IsSkipped(err) {
				color.Yellow(err.Error())
				return reported(err)
			}
			if err != nil {
				var f *exercises.FailedError
				if errors.As(err, &f) {
					color.Red("%s", f.Lines[0])
					for _, l := range f.Lines[1:] {
						fmt.Println("  • " + l)
					}
					return reported(err)
				}
				return err
			}
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}
}
