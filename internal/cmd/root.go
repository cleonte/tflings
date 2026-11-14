package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/engine"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/spf13/cobra"
)

const infoFile = "info.toml"

// newEnv is what running exercises needs. The Terraform engine is looked up
// once: $TFLINGS_TERRAFORM, else the tflings-managed copy, else the first
// terraform on PATH. Floci support is plugged in when the manager is built; until
// then only exercises that need no Floci can run.
func newEnv() *exercises.Env {
	env := &exercises.Env{Embedded: tflings.FS, Solutions: tflings.Solutions}
	managed := os.Getenv("TFLINGS_TERRAFORM")
	if managed == "" {
		managed = filepath.Join(env.HomeDir(), "bin", "terraform")
	}
	if eng, err := engine.Find(context.Background(), managed); err == nil {
		env.Engine = eng
	}
	return env
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
