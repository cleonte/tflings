package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/tui"
	"github.com/cleonte/tflings/internal/workspace"
)

func runTUI(ctx context.Context, changes <-chan []string) error {
	return tui.Run(ctx, tui.Config{
		InfoFile:     infoFile,
		Changes:      changes,
		Env:          newEnv(),
		Source:       tflings.FS,
		Root:         ".",
		Solutions:    tflings.Solutions,
		Editor:       editor(),
		NewExercises: len(workspace.Missing(tflings.FS, ".")),
	})
}

// editor is the editor the learner has set up, for the e key of the
// full-screen view.
func editor() string {
	if e := strings.TrimSpace(os.Getenv("VISUAL")); e != "" {
		return e
	}
	return strings.TrimSpace(os.Getenv("EDITOR"))
}
