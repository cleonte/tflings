package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/cleonte/tflings/internal/engine"
)

// DefaultMaxOutput is how much of a run's stdout and stderr each is kept.
const DefaultMaxOutput = 1 << 20 // 1 MiB

// Call is one run of the engine.
type Call struct {
	Engine engine.Engine
	Dir    string   // working directory (the exercise's workspace folder)
	Env    []string // from Env; nothing else is passed
	Args   []string
	// Timeout is a wall-clock limit for the whole run, children included. It is
	// what stops an offline hang, which AWS_MAX_ATTEMPTS does not (spike 04).
	Timeout time.Duration
	// MaxOutput caps each of stdout and stderr; 0 means DefaultMaxOutput.
	MaxOutput int
}

// Output is what a run produced.
type Output struct {
	Stdout, Stderr []byte
	// Truncated is true when either stream hit the cap.
	Truncated bool
	ExitCode  int
	Duration  time.Duration
}

// TimeoutError means the run was stopped for taking too long.
type TimeoutError struct {
	Args    []string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("terraform %s did not finish in %s and was stopped", firstWord(e.Args), e.Timeout)
}

func firstWord(args []string) string {
	for _, a := range args {
		if len(a) > 0 && a[0] != '-' {
			return a
		}
	}
	return ""
}

// Run runs the engine and waits for it. A non-zero exit code is not an error
// (plan -detailed-exitcode uses 2 for "changes"); failing to start, and the
// timeout, are. When the timeout expires the whole process group is killed, so
// provider processes do not outlive the run.
func Run(ctx context.Context, c Call) (Output, error) {
	if c.Engine == nil {
		return Output{}, errors.New("no engine")
	}
	limit := c.MaxOutput
	if limit <= 0 {
		limit = DefaultMaxOutput
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, c.Engine.Binary(), c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env // never nil: an empty environment, not the parent's
	if cmd.Env == nil {
		cmd.Env = []string{}
	}
	setProcessGroup(cmd)
	cmd.WaitDelay = 3 * time.Second

	stdout := &cappedBuffer{max: limit}
	stderr := &cappedBuffer{max: limit}
	cmd.Stdout, cmd.Stderr = stdout, stderr

	start := time.Now()
	err := cmd.Run()
	out := Output{
		Stdout:    stdout.Bytes(),
		Stderr:    stderr.Bytes(),
		Truncated: stdout.truncated || stderr.truncated,
		Duration:  time.Since(start),
		ExitCode:  -1, // the program did not start, or was killed
	}
	if cmd.ProcessState != nil {
		out.ExitCode = cmd.ProcessState.ExitCode()
	}
	if runCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
		return out, &TimeoutError{Args: c.Args, Timeout: timeout}
	}
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		return out, err
	}
	return out, nil
}

// cappedBuffer keeps the first max bytes and drops the rest, remembering that
// it did. It never fails a write, so the child never blocks on a full pipe.
type cappedBuffer struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	room := b.max - b.buf.Len()
	if room > 0 {
		if len(p) > room {
			b.buf.Write(p[:room])
			b.truncated = true
		} else {
			b.buf.Write(p)
		}
	} else if len(p) > 0 {
		b.truncated = true
	}
	return len(p), nil
}

func (b *cappedBuffer) Bytes() []byte { return b.buf.Bytes() }
