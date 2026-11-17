package exercises

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockWait is how long a command waits for another tflings that is using the
// same exercise's run folder (a `watch` in one terminal, `tflings tf` in
// another) before it gives up with a message. A variable so that tests can
// shorten it.
var lockWait = 2 * time.Minute

const lockPoll = 50 * time.Millisecond

// lockRun takes the lock of an exercise's run folder: an OS file lock, so it
// works between processes as well as between goroutines, and is released by the
// OS if tflings dies. Everything that writes into <home>/runs/<name> (writing
// the files, seeding, init, the Terraform command, forgetting the run) holds
// it, so one tflings cannot delete the folder under another. The lock file sits
// next to the folder, not in it, so that deleting the folder leaves it alone.
func (e *Env) lockRun(ctx context.Context, name string) (unlock func(), err error) {
	return lockFile(ctx, filepath.Join(e.home(), "runs"), name, lockWait)
}

func lockFile(ctx context.Context, runsDir, name string, wait time.Duration) (func(), error) {
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(runsDir, name+".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wait)
	announced := false
	for {
		ok, err := tryLock(f)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		if ok {
			return func() {
				unlockFile(f)
				_ = f.Close()
			}, nil
		}
		if !announced {
			report(ctx, "waiting for another tflings to finish with "+name)
			announced = true
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			_ = f.Close()
			return nil, fmt.Errorf("another tflings is using %s (a `tflings tf`, `watch` or `run` in another terminal). Let it finish, then try again", name)
		}
		time.Sleep(lockPoll)
	}
}
