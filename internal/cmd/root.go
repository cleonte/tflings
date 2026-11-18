package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/engine"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/cleonte/tflings/internal/floci"
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
	mgr := floci.New()
	env.FlociUp = mgr.Healthy
	return env
}

// NewRootCmd builds the tflings command with all its subcommands.
func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "tflings",
		Short:         "Learn Terraform through small broken exercises",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	rootCmd.AddCommand(AwsCmd())
	rootCmd.AddCommand(DoctorCmd(infoFile))
	rootCmd.AddCommand(ExamCmd(infoFile))
	rootCmd.AddCommand(HintCmd(infoFile))
	rootCmd.AddCommand(InitCmd(tflings.FS))
	rootCmd.AddCommand(ListCmd(infoFile))
	rootCmd.AddCommand(ProgressCmd(infoFile))
	rootCmd.AddCommand(ResetCmd(tflings.FS, "."))
	rootCmd.AddCommand(RunCmd(infoFile))
	rootCmd.AddCommand(SandboxCmd())
	rootCmd.AddCommand(SolutionCmd(infoFile, tflings.Solutions))
	rootCmd.AddCommand(TfCmd(infoFile))
	rootCmd.AddCommand(EnvCmd(infoFile))
	rootCmd.AddCommand(UpdateCmd(tflings.FS, "."))
	rootCmd.AddCommand(VerifyCmd(infoFile))
	rootCmd.AddCommand(VersionCmd())
	rootCmd.AddCommand(WatchCmd(infoFile))

	return rootCmd
}

// reportedError marks an error whose message the command has already printed,
// so Execute does not print it again.
type reportedError struct{ err error }

func (r *reportedError) Error() string { return r.err.Error() }
func (r *reportedError) Unwrap() error { return r.err }

// reported wraps err for a command that has already told the user about it.
func reported(err error) error {
	if err == nil {
		return nil
	}
	return &reportedError{err}
}

// printError tells the user about an error that no command has reported: a
// flag or command that does not exist, or a failure with no message of its own.
func printError(w io.Writer, err error) {
	var r *reportedError
	if errors.As(err, &r) {
		return
	}
	fmt.Fprintf(w, "Error: %v\n", err)
	if strings.Contains(err.Error(), "unknown") {
		fmt.Fprintln(w, "Run `tflings --help` to see the commands and flags.")
	}
}

// Execute runs the command line and exits with status 1 on an error.
func Execute(version string) {
	// Ctrl-C stops a run that is still waiting for Terraform or Floci.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := NewRootCmd(version).ExecuteContext(ctx)
	stop()
	if err != nil {
		printError(os.Stderr, err)
		os.Exit(1)
	}
}
