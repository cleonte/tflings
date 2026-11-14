package exercises

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cleonte/tflings/internal/check"
	"github.com/cleonte/tflings/internal/engine"
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

	// Engine runs Terraform. Nil means there is none, and plan, apply, fix and
	// cli exercises are skipped.
	Engine engine.Engine
	// Home is tflings's own folder (scratch runs, plugin cache, CLI config).
	// Empty means a folder under the user's cache directory.
	Home string
	// Workspace is the learner's folder, where info.toml and .tflings.json are.
	// Empty means the current directory.
	Workspace string
}

func (e *Env) workspace() string {
	if e == nil || e.Workspace == "" {
		return "."
	}
	return e.Workspace
}

func (e *Env) home() string {
	if e != nil && e.Home != "" {
		return e.Home
	}
	if h := os.Getenv("TFLINGS_HOME"); h != "" {
		return h
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "tflings")
	}
	return filepath.Join(os.TempDir(), "tflings")
}

// Availability looks at what this machine can run right now.
func (e *Env) Availability(ctx context.Context) Availability {
	var av Availability
	if e == nil {
		return av
	}
	av.Terraform = e.Engine != nil
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
	var res Result
	var err error
	switch e.Mode {
	case ModePlan:
		files, ferr := WorkspaceFiles(os.DirFS(env.workspace()), filepath.ToSlash(e.Path))
		if ferr != nil {
			return Result{}, fmt.Errorf("cannot read %s: %w", e.Path, ferr)
		}
		res, err = e.RunFiles(ctx, env, files)
	default:
		var src []byte
		if e.Mode != ModeFix {
			if src, err = os.ReadFile(e.File()); err != nil {
				return Result{}, fmt.Errorf("cannot read %s: %w", e.File(), err)
			}
		}
		res, err = e.RunSource(ctx, env, src)
	}
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
	case ModePlan:
		return e.runPlan(ctx, env, c, map[string][]byte{ExerciseFile: src})
	}
	return Result{}, fmt.Errorf("%s: %w (%s)", e.Name, ErrNotBuilt, e.Mode)
}

// RunFiles checks an exercise that has several files (plan mode): files are
// keyed by path relative to the exercise folder.
func (e Exercise) RunFiles(ctx context.Context, env *Env, files map[string][]byte) (Result, error) {
	if reason := e.SkipReason(env.Availability(ctx)); reason != "" {
		return Result{}, &SkippedError{fmt.Sprintf("%s %s", e.Name, reason)}
	}
	raw, err := fs.ReadFile(env.Embedded, e.CheckPath())
	if err != nil {
		return Result{}, fmt.Errorf("%s has no check.yaml: %w", e.Name, err)
	}
	c, err := check.Load(raw)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", e.Name, err)
	}
	if e.Mode != ModePlan {
		return Result{}, fmt.Errorf("%s: RunFiles is for plan exercises, not %s", e.Name, e.Mode)
	}
	return e.runPlan(ctx, env, c, files)
}

// runQuiz compares the answer in quiz.toml with the key.
func (e Exercise) runQuiz(c *check.Compiled, src []byte) (Result, error) {
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

// HomeDir is tflings's own folder (see Env.Home), for callers that need to
// look inside it, such as the lookup of a managed Terraform.
func (e *Env) HomeDir() string { return e.home() }
