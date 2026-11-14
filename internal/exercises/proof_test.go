package exercises_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/exercises"
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
	env := &exercises.Env{Embedded: tflings.FS, Solutions: tflings.Solutions}
	for _, ex := range all {
		t.Run(ex.Name, func(t *testing.T) {
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
