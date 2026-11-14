package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/cleonte/tflings/internal/floci"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// flociError prints what to do about a manager error.
func flociError(err error) error {
	switch {
	case errors.Is(err, floci.ErrNoDocker):
		color.Red("Docker is not installed.")
		color.White("tflings runs a small local AWS (Floci) in Docker. Install Docker Desktop, OrbStack, Colima or Docker Engine, then run `tflings aws up`.")
	case errors.Is(err, floci.ErrDockerDown):
		color.Red("Docker is installed, but its daemon is not running.")
		color.White("Start Docker (Docker Desktop, OrbStack, `colima start`, `limactl start`, or `sudo systemctl start docker`), then run `tflings aws up`.")
	case errors.Is(err, floci.ErrPortInUse):
		color.Red("Port %d is already used by another program.", floci.DefaultConfig().Port)
		color.White("tflings uses a port of its own for Floci. Find what holds it with `lsof -i :%d` and stop it, then run `tflings aws up` again.", floci.DefaultConfig().Port)
	case errors.Is(err, floci.ErrNotHealthy):
		color.Red("Floci started but did not answer in time.")
		color.White("Look at its log: `docker logs %s`", floci.DefaultConfig().Name)
	default:
		color.Red("%s", err)
	}
	return err
}

// AwsCmd manages the local AWS (Floci) that apply exercises use.
func AwsCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "aws",
		Short: "Manage the local AWS (Floci) that the exercises use",
		Long: `tflings runs its own Floci container in Docker, on port 4577, and never touches
a Floci or LocalStack you run yourself. Nothing here talks to a real AWS.`,
	}
	mgr := floci.New()
	mgr.Progress = func(s string) { color.White("%s…", s) }

	root.AddCommand(&cobra.Command{
		Use:           "up",
		Short:         "Start Floci (pulls the image the first time)",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgr.Up(cmd.Context()); err != nil {
				return flociError(err)
			}
			color.Green("Floci %s is running on %s", floci.Version, mgr.URL())
			return nil
		},
	})

	var wipe bool
	down := &cobra.Command{
		Use:           "down",
		Short:         "Stop and remove the Floci container (your data stays)",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgr.Down(cmd.Context(), wipe); err != nil {
				return flociError(err)
			}
			if wipe {
				color.Green("Floci is gone, and so is everything the exercises built in it.")
			} else {
				color.Green("Floci is stopped. What the exercises built is kept; `tflings aws up` brings it back.")
			}
			return nil
		},
	}
	down.Flags().BoolVar(&wipe, "wipe", false, "also delete the data volume, that is everything the exercises built")
	root.AddCommand(down)

	root.AddCommand(&cobra.Command{
		Use:           "status",
		Short:         "Say whether Floci is running",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mgr.Healthy(cmd.Context()) {
				color.Green("Floci is running on %s", mgr.URL())
				return nil
			}
			st, err := mgr.Current(cmd.Context())
			switch {
			case err != nil:
				return flociError(err)
			case st.Exists && st.Running:
				color.Yellow("The container runs but Floci does not answer yet (Docker says: %s).", orNone(st.Health))
			case st.Exists:
				color.Yellow("The container exists but is stopped. Run `tflings aws up`.")
			default:
				color.Yellow("Floci is not running. Run `tflings aws up`.")
			}
			return errors.New("floci is not running")
		},
	})

	var yes bool
	reset := &cobra.Command{
		Use:   "reset",
		Short: "Empty Floci: delete everything every exercise has built",
		Long: `Floci has no per-exercise reset, so this empties every fake AWS account at once.
Exercises you have already solved stay solved; an exercise that was in progress
starts again from nothing in Floci. (To start one exercise over, use
tflings reset <exercise>, which gives it a fresh account.)`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("This deletes everything in every exercise's fake AWS account. Continue? [y/N] ")
				line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
					color.White("Nothing was changed.")
					return nil
				}
			}
			if err := mgr.Reset(cmd.Context()); err != nil {
				return flociError(err)
			}
			color.Green("Floci is empty.")
			return nil
		},
	}
	reset.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask")
	root.AddCommand(reset)
	return root
}

func orNone(s string) string {
	if s == "" {
		return "no health status"
	}
	return s
}
