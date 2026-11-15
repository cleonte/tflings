package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// VersionCmd prints what `tflings --version` prints.
func VersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print the version of tflings",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "tflings version %s\n", cmd.Root().Version)
		},
	}
}
