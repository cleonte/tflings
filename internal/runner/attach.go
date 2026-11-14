package runner

import (
	"context"
	"errors"
	"io"
	"os/exec"
)

// Attach runs the engine with the learner's terminal attached: the
// environment is the same sealed one as Run's, but standard input and output are
// the caller's, so that Terraform can ask "Do you want to perform these
// actions?". There is no timeout (the learner is typing); cancelling ctx stops
// it. It returns the exit code.
func Attach(ctx context.Context, c Call, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if c.Engine == nil {
		return -1, errors.New("no engine")
	}
	cmd := exec.CommandContext(ctx, c.Engine.Binary(), c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	if cmd.Env == nil {
		cmd.Env = []string{}
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	if err != nil {
		return -1, err
	}
	return 0, nil
}
