package cmd

import (
	"fmt"

	"github.com/cleonte/tflings/internal/exercises"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ListCmd lists the exercises by topic, with their state.
func ListCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:           "list",
		Short:         "List all exercises, by topic, with their state",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			all, err := exercises.List(infoFile)
			if err != nil {
				return err
			}
			topic, done := "", 0
			for _, ex := range all {
				if ex.Topic != topic {
					topic = ex.Topic
					color.Cyan("\n%s", topic)
				}
				mark, tag := "  ", ""
				if ex.State() == exercises.Done {
					mark = color.GreenString("✓ ")
					done++
				} else if exercises.IsNew(ex.Name) {
					tag = " " + color.YellowString("new")
				}
				fmt.Printf("  %s%-24s %s%s\n", mark, ex.Name, ex.Mode, tag)
			}
			fmt.Printf("\n%d of %d exercises done\n", done, len(all))
			noticeNewExercises()
			return nil
		},
	}
}
