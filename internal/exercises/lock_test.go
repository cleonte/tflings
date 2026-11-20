package exercises

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLockExcludesGoroutines(t *testing.T) {
	dir := t.TempDir()
	unlock, err := lockFile(context.Background(), dir, "x", "ex-x", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lockFile(context.Background(), dir, "x", "ex-x", 200*time.Millisecond); err == nil || !strings.Contains(err.Error(), "another tflings") {
		t.Fatalf("a second lock should be refused with a clear message, got %v", err)
	}
	// Another exercise's lock is independent.
	other, err := lockFile(context.Background(), dir, "y", "ex-y", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	other()

	// A waiter gets the lock once the holder lets go.
	var wg sync.WaitGroup
	wg.Add(1)
	got := make(chan error, 1)
	go func() {
		defer wg.Done()
		u, err := lockFile(context.Background(), dir, "x", "ex-x", 5*time.Second)
		if err == nil {
			u()
		}
		got <- err
	}()
	time.Sleep(150 * time.Millisecond)
	unlock()
	wg.Wait()
	if err := <-got; err != nil {
		t.Errorf("the waiter should get the lock: %v", err)
	}
}

// The lock must hold between processes: the test runs itself as a child that
// takes the lock and sleeps.
func TestLockExcludesProcesses(t *testing.T) {
	if dir := os.Getenv("TFLINGS_LOCK_HELPER"); dir != "" {
		u, err := lockFile(context.Background(), dir, "x", "ex-x", time.Second)
		if err != nil {
			os.Exit(3)
		}
		_ = os.WriteFile(filepath.Join(dir, "held"), nil, 0o644)
		time.Sleep(1500 * time.Millisecond)
		u()
		return
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockExcludesProcesses$")
	cmd.Env = append(os.Environ(), "TFLINGS_LOCK_HELPER="+dir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Wait() }()
	for i := 0; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, "held")); err == nil {
			break
		}
		if i > 100 {
			t.Fatal("the child never took the lock")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := lockFile(context.Background(), dir, "x", "ex-x", 100*time.Millisecond); err == nil {
		t.Fatal("the lock held by another process should be refused")
	}
	u, err := lockFile(context.Background(), dir, "x", "ex-x", 5*time.Second)
	if err != nil {
		t.Fatalf("after the child exits the lock should be free: %v", err)
	}
	u()
}

func TestForgetRunRespectsTheLock(t *testing.T) {
	env := &Env{Home: t.TempDir()}
	run := filepath.Join(env.Home, "runs", "x", "src")
	if err := os.MkdirAll(run, 0o755); err != nil {
		t.Fatal(err)
	}
	unlock, err := env.lockRun(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	old := forgetWait
	forgetWait = 200 * time.Millisecond
	defer func() { forgetWait = old }()
	if err := env.ForgetRun("x"); err == nil || !strings.Contains(err.Error(), "another tflings") {
		t.Fatalf("forgetting a run that is in use should be refused, got %v", err)
	}
	if _, err := os.Stat(run); err != nil {
		t.Fatal("the folder must not be deleted under a running command")
	}
	unlock()
	if err := env.ForgetRun("x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(run); err == nil {
		t.Error("the folder should be gone")
	}
}

// A learner reads the exercise's name in the message, not the internal key of
// its folder (which carries the folder's salt).
func TestLockMessageNamesTheExercise(t *testing.T) {
	dir := t.TempDir()
	unlock, err := lockFile(context.Background(), dir, "e702b45f-locking1", "locking1", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	_, err = lockFile(context.Background(), dir, "e702b45f-locking1", "locking1", 100*time.Millisecond)
	if err == nil {
		t.Fatal("the second lock should be refused")
	}
	if strings.Contains(err.Error(), "e702b45f") || !strings.Contains(err.Error(), "using locking1 ") {
		t.Errorf("the message should name the exercise only, got %q", err)
	}
}
