package exercises

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/cleonte/tflings/internal/check"
	"github.com/pelletier/go-toml/v2"
)

// Env is what running exercises needs.
type Env struct {
	Embedded  fs.FS // the tflings.FS: info.toml and exercises/
	Solutions fs.FS // the embedded solutions/ tree
	// TerraformFound and FlociUp say what this machine has. They are functions
	// so that `list` and `doctor` can ask without paying for it every time. Nil
	// means "no".
	TerraformFound func() bool
	FlociUp        func(ctx context.Context) bool
	HCPToken       func() bool
}

// Availability looks at what this machine can run right now.
func (e *Env) Availability(ctx context.Context) Availability {
	var av Availability
	if e == nil {
		return av
	}
	if e.TerraformFound != nil {
		av.Terraform = e.TerraformFound()
	}
	if e.FlociUp != nil {
		av.Floci = e.FlociUp(ctx)
	}
	if e.HCPToken != nil {
		av.HCP = e.HCPToken()
	}
	return av
}

// Result is what a successful run reports.
type Result struct {
	Exercise Exercise
	// Checked is how many assertions passed (1 for a quiz).
	Checked int
	// Explanation is shown after a pass, when the exercise has one.
	Explanation string
}

// FailedError means the exercise is not solved. Lines are written for the
// learner, one problem per line.
type FailedError struct {
	Lines []string
	// Details is extra context for a learner who is stuck: not problems
	// themselves, so it is shown without bullets.
	Details []string
}

func (e *FailedError) Error() string { return strings.Join(e.Lines, "\n") }

// SkippedError means the exercise cannot run here, and says why.
type SkippedError struct{ Reason string }

func (e *SkippedError) Error() string { return e.Reason }

func IsSkipped(err error) bool {
	var s *SkippedError
	return errors.As(err, &s)
}

// ErrNotBuilt is returned for a mode whose runner does not exist yet.
var ErrNotBuilt = errors.New("this exercise mode is not built yet")

// Run checks the learner's exercise file. It does not look at the
// "I AM NOT DONE" marker; callers decide what a pending-but-passing exercise
// means.
func (e Exercise) Run(ctx context.Context, env *Env) (Result, error) {
	if reason := e.SkipReason(env.Availability(ctx)); reason != "" {
		return Result{}, &SkippedError{fmt.Sprintf("%s %s", e.Name, reason)}
	}
	var src []byte
	if e.Mode != ModeFix {
		var err error
		if src, err = os.ReadFile(e.File()); err != nil {
			return Result{}, fmt.Errorf("cannot read %s: %w", e.File(), err)
		}
	}
	res, err := e.RunSource(ctx, env, src)
	if err == nil {
		if err := markPassed(e.Name); err != nil {
			return Result{}, fmt.Errorf("it passes, but cannot save your progress in %s: %w", ProgressFile, err)
		}
	}
	return res, err
}

// RunSource checks src as if it were the content of the exercise file.
func (e Exercise) RunSource(ctx context.Context, env *Env, src []byte) (Result, error) {
	raw, err := fs.ReadFile(env.Embedded, e.CheckPath())
	if err != nil {
		return Result{}, fmt.Errorf("%s has no check.yaml: %w", e.Name, err)
	}
	c, err := check.Load(raw)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", e.Name, err)
	}
	switch e.Mode {
	case ModeQuiz:
		return e.runQuiz(c, src)
	}
	return Result{}, fmt.Errorf("%s: %w (%s)", e.Name, ErrNotBuilt, e.Mode)
}

// runQuiz compares the answer in quiz.toml with the key.
func (e Exercise) runQuiz(c *check.Check, src []byte) (Result, error) {
	var q struct {
		Answer any `toml:"answer"`
	}
	if err := toml.Unmarshal(src, &q); err != nil {
		return Result{}, &FailedError{Lines: []string{"quiz.toml could not be read: " + err.Error()}}
	}
	var given []string
	switch a := q.Answer.(type) {
	case string:
		for _, p := range strings.FieldsFunc(a, func(r rune) bool { return r == ',' || r == ' ' }) {
			given = append(given, p)
		}
	case []any:
		for _, v := range a {
			if s, ok := v.(string); ok {
				given = append(given, s)
			}
		}
	}
	if len(given) == 0 {
		return Result{}, &FailedError{Lines: []string{`Set answer = "..." in quiz.toml to the letter of your choice.`}}
	}
	if !c.QuizOK(given) {
		return Result{}, &FailedError{Lines: []string{"That is not the right answer. Read the question again."}}
	}
	return Result{Exercise: e, Checked: 1, Explanation: c.Explanation}, nil
}
