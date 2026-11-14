package exercises_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/engine"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/cleonte/tflings/internal/floci"
)

// TestEveryExerciseIsProven is the course's core invariant: the file as
// shipped does not pass, and the solution does. Modes that need Terraform or
// Floci are proven by their own runners (they are skipped here until built).
func TestEveryExerciseIsProven(t *testing.T) {
	raw, err := fs.ReadFile(tflings.FS, "info.toml")
	if err != nil {
		t.Fatal(err)
	}
	all, err := exercises.ParseInfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := &exercises.Env{Embedded: tflings.FS, Solutions: tflings.Solutions, Home: t.TempDir(), Workspace: t.TempDir()}
	// A run never meets what an earlier test run left in Floci.
	env.AccountSalt = strconv.FormatInt(time.Now().UnixNano(), 36)
	if eng, err := engine.Find(context.Background(), os.Getenv("TFLINGS_TEST_TERRAFORM")); err == nil {
		env.Engine = eng
	}
	// Exercises that use the AWS provider need Floci running and a home whose
	// plugin cache holds the provider (`tflings doctor --prefetch` with
	// TFLINGS_HOME set to it); CI prepares both.
	flociHome := os.Getenv("TFLINGS_TEST_HOME")
	mgr := floci.New()
	flociReady := flociHome != "" && mgr.Healthy(context.Background())
	env.FlociUp = func(context.Context) bool { return flociReady }
	if flociReady {
		env.Home = flociHome
	}
	for _, ex := range all {
		t.Run(ex.Name, func(t *testing.T) {
			if ex.NeedsFloci() && !flociReady {
				t.Skip("needs Floci running and TFLINGS_TEST_HOME (a home with the AWS provider prefetched)")
			}
			if ex.Mode == exercises.ModePlan || ex.Mode == exercises.ModeApply {
				proveWithFiles(t, env, ex)
				return
			}
			if ex.Mode != exercises.ModeQuiz {
				t.Skipf("mode %s is proven by its own runner", ex.Mode)
			}
			broken, err := fs.ReadFile(tflings.FS, ex.Source())
			if err != nil {
				t.Fatalf("no exercise file: %v", err)
			}
			if _, err := ex.RunSource(context.Background(), env, broken); err == nil {
				t.Error("the exercise as shipped should not pass")
			} else if !errors.As(err, new(*exercises.FailedError)) {
				t.Errorf("expected a FailedError, got %v", err)
			}
			solution, err := fs.ReadFile(tflings.Solutions, ex.SolutionPath())
			if err != nil {
				t.Fatalf("no solution: %v", err)
			}
			if _, err := ex.RunSource(context.Background(), env, solution); err != nil {
				t.Errorf("the solution should pass: %v", err)
			}
			for _, p := range []string{ex.CheckPath()} {
				if _, err := fs.Stat(tflings.FS, p); err != nil {
					t.Errorf("missing %s", p)
				}
			}
		})
	}
}

// proveWithFiles is the proof for plan exercises: the folder as shipped fails
// with a message for the learner, and the folder with the solution in place
// passes. It needs a Terraform (set TFLINGS_TEST_TERRAFORM, or have one on
// PATH); CI installs one.
func proveWithFiles(t *testing.T, env *exercises.Env, ex exercises.Exercise) {
	t.Helper()
	if env.Engine == nil {
		t.Skip("no Terraform found: set TFLINGS_TEST_TERRAFORM")
	}
	for _, p := range []string{ex.CheckPath(), ex.Source(), ex.SolutionPath()} {
		var fsys fs.FS = tflings.FS
		if p == ex.SolutionPath() {
			fsys = tflings.Solutions
		}
		if _, err := fs.Stat(fsys, p); err != nil {
			t.Fatalf("missing %s", p)
		}
	}
	broken, err := exercises.WorkspaceFiles(tflings.FS, ex.Path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ex.RunFiles(context.Background(), env, broken)
	var failed *exercises.FailedError
	if err == nil {
		t.Fatal("the exercise as shipped should not pass")
	}
	if !errors.As(err, &failed) {
		t.Fatalf("the shipped exercise should fail with a message for the learner, got: %v", err)
	}
	solved, err := ex.ProofFiles(env)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ex.RunFiles(context.Background(), env, solved); err != nil {
		var f *exercises.FailedError
		if errors.As(err, &f) {
			t.Fatalf("the solution should pass:\n%v\n%v", f.Lines, f.Details)
		}
		t.Fatalf("the solution should pass: %v", err)
	}
}

// TestShortcutsAreRefused tries to cheat: answers that hard-code the value the
// first plan checks. The variants of check.yaml plan again with other input,
// so these must fail.
func TestShortcutsAreRefused(t *testing.T) {
	eng, err := engine.Find(context.Background(), os.Getenv("TFLINGS_TEST_TERRAFORM"))
	if err != nil {
		t.Skip("no Terraform found: set TFLINGS_TEST_TERRAFORM")
	}
	env := &exercises.Env{Embedded: tflings.FS, Solutions: tflings.Solutions, Home: t.TempDir(), Workspace: t.TempDir(), Engine: eng}
	cheats := map[string]string{
		"expressions1": "variable \"environment\" {\n  type    = string\n  default = \"prod\"\n}\n\nlocals {\n  replicas = 3\n}\n\noutput \"replicas\" {\n  value = local.replicas\n}\n",
		"functions1":   "variable \"name\" {\n  type    = string\n  default = \"tflings\"\n}\n\noutput \"shout\" {\n  value = \"TFLINGS\"\n}\n",
		"variables1":   "variable \"environment\" {\n  type        = string\n  description = \"Which environment this is.\"\n  default     = \"dev\"\n}\n\noutput \"name\" {\n  value = \"app-dev\"\n}\n",
		"variables2":   "variable \"instance_count\" {\n  type    = number\n  default = 3\n}\n\noutput \"total_cost\" {\n  value = 30\n}\n",
	}
	raw, _ := fs.ReadFile(tflings.FS, "info.toml")
	all, _ := exercises.ParseInfo(raw)
	for _, ex := range all {
		cheat, ok := cheats[ex.Name]
		if !ok {
			continue
		}
		t.Run(ex.Name, func(t *testing.T) {
			_, err := ex.RunFiles(context.Background(), env, map[string][]byte{"main.tf": []byte(cheat)})
			var failed *exercises.FailedError
			if !errors.As(err, &failed) {
				t.Fatalf("the shortcut must be refused with a message, got %v", err)
			}
		})
	}
}

// A variant that builds the configuration first (apply_first) must leave no
// state behind: the next run, and the next variant, start from nothing.
func TestVariantsLeaveNoState(t *testing.T) {
	eng, err := engine.Find(context.Background(), os.Getenv("TFLINGS_TEST_TERRAFORM"))
	if err != nil {
		t.Skip("no Terraform found: set TFLINGS_TEST_TERRAFORM")
	}
	home := t.TempDir()
	env := &exercises.Env{Embedded: tflings.FS, Solutions: tflings.Solutions, Home: home, Workspace: t.TempDir(), Engine: eng}
	raw, _ := fs.ReadFile(tflings.FS, "info.toml")
	all, _ := exercises.ParseInfo(raw)
	for _, ex := range all {
		if ex.Name != "lifecycle1" {
			continue
		}
		files, err := ex.ProofFiles(env)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ex.RunFiles(context.Background(), env, files); err != nil {
			t.Fatal(err)
		}
		for _, n := range []string{"terraform.tfstate", "terraform.tfstate.backup"} {
			if _, err := os.Stat(home + "/runs/lifecycle1/src/" + n); err == nil {
				t.Errorf("%s was left behind by a variant", n)
			}
		}
	}
}
