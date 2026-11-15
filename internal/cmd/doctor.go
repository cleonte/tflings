package cmd

import (
	"errors"
	"fmt"

	"github.com/cleonte/tflings/internal/doctor"
	"github.com/cleonte/tflings/internal/floci"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// DoctorCmd checks the machine and says how to fix what is missing.
func DoctorCmd(infoFile string) *cobra.Command {
	var prefetch bool
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Check that this machine can run the course, and say how to fix what cannot",
		Long: `Checks Terraform, Docker, Floci, the AWS provider cache and whether
<account>.localhost resolves. Quizzes and Terraform-only exercises need only
Terraform; exercises that build things need Docker and Floci too.

With --prefetch it downloads the AWS provider (about 250 MB) into tflings's
cache, once, so that exercises can start with no network afterwards.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			env := newEnv()
			if prefetch {
				if err := doctor.Prefetch(cmd.Context(), env, func(s string) { color.White("%s…", s) }); err != nil {
					color.Red("%s", err)
					return reported(err)
				}
				color.Green("The AWS provider is in the cache.\n")
			}
			results := doctor.Run(cmd.Context(), doctor.Options{
				Env: env, Floci: floci.New(), InfoFile: infoFile, Workspace: ".",
			})
			for _, r := range results {
				switch r.Level {
				case doctor.OK:
					color.Green("✓ %s: %s", r.Name, r.Detail)
				case doctor.Warn:
					color.Yellow("! %s: %s", r.Name, r.Detail)
				default:
					color.Red("✗ %s: %s", r.Name, r.Detail)
				}
				if r.Fix != "" {
					fmt.Printf("    fix: %s\n", r.Fix)
				}
			}
			if doctor.Worst(results) == doctor.Fail {
				return errors.New("something that is needed is missing")
			}
			return nil
		},
	}
	c.Flags().BoolVar(&prefetch, "prefetch", false, "download the AWS provider into the cache (needs the network, once)")
	return c
}
