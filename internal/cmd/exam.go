package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/exam"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/spf13/cobra"
)

// ExamCmd is the mock exam: multiple choice questions from the embedded bank.
func ExamCmd(infoFile string) *cobra.Command {
	var count, minutes int
	var objective string
	var practice, weak bool
	var seed int64
	cmd := &cobra.Command{
		Use:   "exam",
		Short: "Mock exam: multiple choice questions, scored per objective",
		Long: `Asks multiple choice questions from the question bank, in a random order.
Nothing is said about the answers until the end (use --practice to see the
answer after each question). Some questions ask for two answers, like the
real exam: type them like a,c.

HashiCorp does not publish the number of questions or the pass mark of the
Terraform Associate exam, so the defaults here (30 questions, 60 minutes) are
ours, and the score is only a guide.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, err := exam.Load(tflings.Questions, "questions")
			if err != nil {
				return err
			}
			home := (&exercises.Env{}).HomeDir()
			qs := exam.Filter(all, objective)
			if weak {
				hist, _ := exam.LoadHistory(home)
				ids := exam.Weakest(hist, qs, 3)
				qs = exam.FilterSet(qs, ids)
				fmt.Fprintf(cmd.OutOrStdout(), "Weakest objectives (never asked first): %s\n", strings.Join(ids, ", "))
			}
			if len(qs) == 0 {
				return fmt.Errorf("no questions for objective %q", objective)
			}
			if seed == 0 {
				seed = time.Now().UnixNano()
			}
			o := exam.Options{Count: count, Practice: practice, Seed: seed, Exercises: exercisesByObjective()}
			if minutes > 0 && !practice {
				o.Duration = time.Duration(minutes) * time.Minute
			}
			n := min(count, len(qs))
			fmt.Fprintf(cmd.OutOrStdout(), "Mock exam: %d questions", n)
			if o.Duration > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), ", %d minutes (time is checked when a question is shown)", minutes)
			}
			fmt.Fprintln(cmd.OutOrStdout(), ".")
			res := exam.Sit(os.Stdin, cmd.OutOrStdout(), qs, o)
			exam.Report(cmd.OutOrStdout(), res, o)
			if err := exam.Append(home, exam.RecordOf(res, practice, time.Now())); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "could not save the result: %v\n", err)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&count, "count", "n", 30, "number of questions")
	cmd.Flags().IntVar(&minutes, "minutes", 60, "time limit in minutes (0: none; not used with --practice)")
	cmd.Flags().StringVar(&objective, "objective", "", "only questions of one objective (4) or sub-objective (4g)")
	cmd.Flags().BoolVar(&practice, "practice", false, "show the answer and the explanation after each question")
	cmd.Flags().BoolVar(&weak, "weak", false, "ask about the three objectives you score lowest on (see tflings progress)")
	cmd.Flags().Int64Var(&seed, "seed", 0, "random seed, to ask the same questions again")
	return cmd
}

// exercisesByObjective lists, for each objective, the exercises that practise it.
func exercisesByObjective() map[string][]string {
	raw, err := tflings.FS.ReadFile("info.toml")
	if err != nil {
		return nil
	}
	all, err := exercises.ParseInfo(raw)
	if err != nil {
		return nil
	}
	m := map[string][]string{}
	for _, ex := range all {
		m[ex.Objective] = append(m[ex.Objective], ex.Name)
	}
	return m
}
