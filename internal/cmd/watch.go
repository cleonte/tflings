package cmd

import (
	"bufio"
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/cleonte/tflings/internal/exercises"
	"github.com/cleonte/tflings/internal/watcher"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// debounce is how long the watcher waits for a burst of file events to settle.
const debounce = 150 * time.Millisecond

func WatchCmd(infoFile string) *cobra.Command {
	var plain bool

	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Check exercises when files are saved",
		Long: `Checks the next exercise whenever you save a file.

Prints line-by-line output. (The full-screen view comes later.)

Type a command and press Enter: h (hint), l (list), r (run
again) or q (quit).`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			root := filepath.Join(filepath.Dir(infoFile), "exercises")
			changes, err := watcher.Watch(ctx, root, debounce)
			if err != nil {
				color.Red("Can't watch %s: %s", root, err)
				color.Yellow("Run tflings from the folder that holds the exercises, or create one with `tflings init`.")
				return err
			}

			noticeNewExercises()
			watchLoop(cmd.InOrStdin(), changes, watchActions{
				run:  func() { _ = runNext(ctx, infoFile) },
				list: func() { _ = ListCmd(infoFile).RunE(nil, nil) },
				hint: func() { printNextHint(ctx, infoFile) },
			})
			return nil
		},
	}
	watchCmd.Flags().BoolVar(&plain, "plain", false, "line-by-line output (the full-screen view is not built yet, so this is the only mode)")
	return watchCmd
}

func printNextHint(ctx context.Context, infoFile string) {
	ex, err := exercises.NextPending(infoFile, newEnv().Availability(ctx))
	if err != nil {
		color.Yellow("There is no pending exercise to give a hint for.")
		return
	}
	color.Cyan(ex.Hint)
	if ex.Docs != "" {
		color.White("\nDocs: %s", ex.Docs)
	}
}

type watchActions struct {
	run, list, hint func()
}

// watchLoop runs the next exercise now and whenever files change, and answers
// the commands typed by the learner, until they quit or input ends. Everything
// happens in this one loop, so a run never overlaps with another run or with a
// command.
func watchLoop(in io.Reader, changes <-chan []string, actions watchActions) {
	actions.run()

	lines := make(chan string)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			lines <- strings.TrimSpace(scanner.Text())
		}
	}()

	for {
		select {
		case _, ok := <-changes:
			if !ok {
				return
			}
			actions.run()

		case line, ok := <-lines:
			if !ok {
				return
			}
			switch line {
			case "l", "list":
				actions.list()
			case "h", "hint":
				actions.hint()
			case "r", "run":
				actions.run()
			case "q", "quit", "exit":
				color.Green("Bye from tflings o/")
				return
			default:
				color.Yellow("commands: h (hint), l (list), r (run again), q (quit)")
			}
		}
	}
}
