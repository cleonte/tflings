package cmd

import (
	"fmt"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/exam"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ProgressCmd shows, per exam objective, the exercises done and the mock exam score.
func ProgressCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:   "progress",
		Short: "Readiness per exam objective: exercises done and mock exam score",
		Long: `For each objective of the Terraform Associate 004 content list, shows how many
of its exercises are done (run it inside your exercises folder) and how you score
on its mock exam questions (tflings exam). Objectives marked with ! need work:
no exercise done yet, or under 70% on the questions you have been asked.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			total, done := map[string]int{}, map[string]int{}
			all, err := exercises.List(infoFile)
			local := err == nil
			if !local {
				raw, rerr := tflings.FS.ReadFile("info.toml")
				if rerr != nil {
					return rerr
				}
				if all, err = exercises.ParseInfo(raw); err != nil {
					return err
				}
			}
			for _, ex := range all {
				total[ex.Objective]++
				if local && ex.State() == exercises.Done {
					done[ex.Objective]++
				}
			}
			hist, _ := exam.LoadHistory((&exercises.Env{}).HomeDir())
			tot := exam.Totals(hist)

			fmt.Printf("%-3s %-52s %-10s %s\n", "", "Objective", "Exercises", "Exam questions")
			needs := 0
			for _, o := range exam.Objectives {
				ex := "-"
				if total[o.ID] > 0 {
					ex = fmt.Sprintf("%d of %d", done[o.ID], total[o.ID])
				}
				q := "not asked"
				t := tot[o.ID]
				if t.N > 0 {
					q = fmt.Sprintf("%d of %d (%d%%)", t.Right, t.N, t.Right*100/t.N)
				}
				weak := (local && total[o.ID] > 0 && done[o.ID] == 0) || (t.N > 0 && t.Right*100/t.N < 70)
				mark := "   "
				if weak {
					mark = color.RedString("!  ")
					needs++
				}
				fmt.Printf("%s%-4s%-48s %-10s %s\n", mark, o.ID, trunc(o.Title, 47), ex, q)
			}
			if !local {
				fmt.Println("\nRun this inside your exercises folder to see the exercises you have done.")
			}
			if needs > 0 {
				fmt.Printf("\n%d objectives need work. `tflings exam --weak` asks about the lowest ones.\n", needs)
			}
			return nil
		},
	}
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
