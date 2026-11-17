package cmd

import (
	"errors"
	"fmt"

	"github.com/cleonte/tflings/internal/doctor"
	"github.com/cleonte/tflings/internal/floci"
	"github.com/cleonte/tflings/internal/install"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// DoctorCmd checks the machine and says how to fix what is missing.
func DoctorCmd(infoFile string) *cobra.Command {
	var prefetch, installTF bool
	var tfVersion string
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Check that this machine can run the course, and say how to fix what cannot",
		Long: `Checks Terraform, Docker, Floci, the AWS provider cache and whether
<account>.localhost resolves. Quizzes and Terraform-only exercises need only
Terraform; exercises that build things need Docker and Floci too.

With --prefetch it downloads the AWS provider (about 250 MB) into tflings's
cache, once, so that exercises can start with no network afterwards.

With --install-terraform it downloads Terraform from releases.hashicorp.com into
tflings's own folder (nothing system-wide), checking the release's checksums and
their signature. The default is the newest 1.12.x, the version the exam tests;
--terraform-version 1.12.3 or a constraint such as "~> 1.12.0" chooses another.
It is used before any terraform on your PATH.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env := newEnv()
			if installTF {
				color.White("Downloading Terraform from releases.hashicorp.com…")
				path, ver, err := install.Terraform(cmd.Context(), env.HomeDir(), tfVersion, install.Download)
				if err != nil {
					color.Red("%s", err)
					return reported(err)
				}
				color.Green("Installed Terraform %s at %s\n", ver, path)
				env = newEnv() // so that the checks below see it
			}
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
	c.Flags().BoolVar(&installTF, "install-terraform", false, "download Terraform into tflings's folder (needs the network)")
	c.Flags().StringVar(&tfVersion, "terraform-version", "", "with --install-terraform: a version (1.12.3) or a constraint (\"~> 1.12.0\"); default the newest 1.12.x")
	c.Flags().BoolVar(&prefetch, "prefetch", false, "download the AWS provider into the cache (needs the network, once)")
	return c
}
