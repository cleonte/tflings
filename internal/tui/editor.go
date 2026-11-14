package tui

import (
	"os/exec"
	"strings"
)

// editorCommand is the command that opens file in the editor the learner set
// (e.g. "code -w" from $VISUAL or $EDITOR). It returns nil when no editor is set.
// It has nothing to do with Terraform: this view runs it with the terminal
// handed over.
func editorCommand(editor, file string) *exec.Cmd {
	fields := strings.Fields(editor)
	if len(fields) == 0 {
		return nil
	}
	return exec.Command(fields[0], append(fields[1:], file)...)
}
