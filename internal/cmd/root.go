package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/spf13/cobra"
)

const infoFile = "info.toml"

// newEnv is what running exercises needs. Terraform and Floci support are
// plugged in as they get built; until then only quiz exercises can run.
func newEnv() *exercises.Env {
	return &exercises.Env{Embedded: tflings.FS, Solutions: tflings.Solutions}
}

func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "tflings",
		Short:         "Learn Terraform through small broken exercises",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	rootCmd.AddCommand(HintCmd(infoFile))
	rootCmd.AddCommand(InitCmd(tflings.FS))
	rootCmd.AddCommand(ListCmd(infoFile))
	rootCmd.AddCommand(ResetCmd(tflings.FS, "."))
	rootCmd.AddCommand(RunCmd(infoFile))
	rootCmd.AddCommand(SolutionCmd(infoFile, tflings.Solutions))
	rootCmd.AddCommand(UpdateCmd(tflings.FS, "."))
	rootCmd.AddCommand(VerifyCmd(infoFile))
	rootCmd.AddCommand(WatchCmd(infoFile))

	return rootCmd
}

func Execute(version string) {
	// Ctrl-C stops a run that is still waiting for Terraform or Floci.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := NewRootCmd(version).ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
