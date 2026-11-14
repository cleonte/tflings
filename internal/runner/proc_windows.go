//go:build windows

package runner

import "os/exec"

// setProcessGroup is a stub: killing the child is all that is done on Windows
// for now (killing a whole process tree there is untested work for later).
func setProcessGroup(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return cmd.Process.Kill()
	}
}
