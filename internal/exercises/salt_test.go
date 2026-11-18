package exercises

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cleonte/tflings/internal/runner"
)

func TestFoldersWithoutASaltKeepTheirNames(t *testing.T) {
	ws := t.TempDir()
	env := &Env{Home: t.TempDir(), Workspace: ws}
	ex := Exercise{Name: "buckets1"}
	r, err := env.runFor(ex)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(env.Home, "runs", "buckets1"); r.dir != want {
		t.Errorf("run folder = %s, want %s", r.dir, want)
	}
	if got, want := AccountFor(ws, "buckets1", ""), runner.Account("buckets1", 0); got != want {
		t.Errorf("account = %s, want the unsalted %s", got, want)
	}
}

func TestTwoFoldersDoNotShareRunsOrAccounts(t *testing.T) {
	home := t.TempDir()
	a, b := t.TempDir(), t.TempDir()
	for _, d := range []string{a, b} {
		if err := EnsureSalt(d); err != nil {
			t.Fatal(err)
		}
	}
	if Salt(a) == "" || Salt(a) == Salt(b) {
		t.Fatalf("each folder needs its own salt: %q %q", Salt(a), Salt(b))
	}
	// Asking again does not change it.
	s := Salt(a)
	if err := EnsureSalt(a); err != nil || Salt(a) != s {
		t.Fatalf("salt changed: %q -> %q (%v)", s, Salt(a), err)
	}
	ex := Exercise{Name: "buckets1", Mode: ModeFix}
	envA := &Env{Home: home, Workspace: a}
	envB := &Env{Home: home, Workspace: b}
	ra, err := envA.runFor(ex)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := envB.runFor(ex)
	if err != nil {
		t.Fatal(err)
	}
	if ra.dir == rb.dir {
		t.Fatal("two folders share a run folder")
	}
	if AccountFor(a, "buckets1", "") == AccountFor(b, "buckets1", "") {
		t.Fatal("two folders share a fake account")
	}
	for _, d := range []string{ra.src, rb.src} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Reset in folder A leaves folder B's state alone.
	if err := envA.ForgetRun("buckets1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ra.src); err == nil {
		t.Error("A's run should be gone")
	}
	if _, err := os.Stat(rb.src); err != nil {
		t.Errorf("B's run must survive A's reset: %v", err)
	}
	// A reset counts per folder.
	if err := RecordReset(a, "buckets1"); err != nil {
		t.Fatal(err)
	}
	if ResetCount(b, "buckets1") != 0 || ResetCount(a, "buckets1") != 1 {
		t.Error("the reset counter is per folder")
	}
}

// A pass is recorded in the progress file of the env's workspace, the one the
// reset counter lives in, not in the current directory.
func TestMarkPassedUsesTheWorkspace(t *testing.T) {
	ws := t.TempDir()
	if err := markPassedIn(ws, "x1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := readProgressIn(ws).Done["x1"]; !ok {
		t.Error("the pass should be in the workspace's file")
	}
	if _, err := os.Stat(ProgressFile); err == nil {
		if _, ok := readProgressIn(".").Done["x1"]; ok {
			t.Error("the pass leaked into the current directory")
		}
	}
}
