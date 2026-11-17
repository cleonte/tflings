package exercises

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cleonte/tflings/internal/awscheck"
	"github.com/cleonte/tflings/internal/check"
	"github.com/cleonte/tflings/internal/runner"
	"sigs.k8s.io/yaml"
)

// Fix mode. A fix exercise is a situation that has already gone wrong: tflings
// builds it once (seeding), the learner repairs it with Terraform commands, run
// through `tflings tf` so that they use the sealed environment, and `tflings run`
// checks the end state.
//
//	exercises/<topic>/<name>/
//	  README.md        the ticket
//	  main.tf ...      the configuration as it is now (the learner may edit it)
//	  check.yaml       embedded only
//	  setup/steps.yaml embedded only: how the situation is built
//	  setup/<dir>/     embedded only: Terraform files that a step applies
//	solutions/<topic>/<name>/
//	  solution.sh      Terraform commands, one per line
//	  *.tf             files the solution changes
//
// The state lives in the run folder, not in the learner's folder, and survives
// between commands, like an apply exercise's. `reset` forgets it, so the next
// command builds the situation again in a fresh account.

// StepsFile is the file of a fix exercise that says how to build the situation.
const StepsFile = "steps.yaml"

// seededMarker is the file in a run folder that says the situation was built.
const seededMarker = "seeded"

// seededInFloci is what the marker holds when the situation was built in Floci.
const seededInFloci = "floci"

// seedStateFile keeps the state right after seeding, for `seed` in check.yaml.
const seedStateFile = "seed-state.json"

// SetupSteps is the setup/steps.yaml of a fix exercise: the steps that build the
// situation the learner has to repair.
type SetupSteps struct {
	Steps []setupStep `json:"steps"`
}

// setupStep is exactly one of its fields.
type setupStep struct {
	// Apply names a folder inside setup/ whose Terraform files replace the
	// configuration, then init and apply are run.
	Apply string `json:"apply,omitempty"`
	// Mutate changes something in Floci behind Terraform's back.
	Mutate *setupMutation `json:"mutate,omitempty"`
}

type setupMutation struct {
	Call string         `json:"call"`
	Args map[string]any `json:"args"`
}

// LoadSteps reads and validates the setup of a fix exercise from the embedded tree.
func (e Exercise) LoadSteps(fsys fs.FS) (SetupSteps, error) {
	var s SetupSteps
	raw, err := fs.ReadFile(fsys, path.Join(e.SetupPath(), StepsFile))
	if err != nil {
		return s, fmt.Errorf("%s has no %s: %w", e.Name, path.Join(SetupDir, StepsFile), err)
	}
	if err := yaml.UnmarshalStrict(raw, &s); err != nil {
		return s, fmt.Errorf("%s: %s: %w", e.Name, StepsFile, err)
	}
	if len(s.Steps) == 0 {
		return s, fmt.Errorf("%s: %s has no steps", e.Name, StepsFile)
	}
	for i, st := range s.Steps {
		switch {
		case st.Apply != "" && st.Mutate == nil:
			if _, err := fs.Stat(fsys, path.Join(e.SetupPath(), st.Apply)); err != nil {
				return s, fmt.Errorf("%s: step %d: no folder setup/%s", e.Name, i+1, st.Apply)
			}
		case st.Mutate != nil && st.Apply == "":
			if err := awscheck.ValidateMutation(st.Mutate.Call, st.Mutate.Args); err != nil {
				return s, fmt.Errorf("%s: step %d: %w", e.Name, i+1, err)
			}
			if !e.NeedsFloci() {
				return s, fmt.Errorf("%s: step %d changes Floci, so the exercise must say requires = [\"floci\"]", e.Name, i+1)
			}
		default:
			return s, fmt.Errorf("%s: step %d must have exactly one of apply, mutate", e.Name, i+1)
		}
	}
	return s, nil
}

// rawFiles reads every file under dir, keyed by path relative to it. Unlike
// WorkspaceFiles it keeps README.md and embedded-only files: it reads setup
// folders, which are embedded-only by definition.
func rawFiles(fsys fs.FS, dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		files[strings.TrimPrefix(strings.TrimPrefix(p, dir), "/")] = data
		return nil
	})
	return files, err
}

// initMode says what a failing init means.
type initMode int

const (
	// initStrict: the check wants an initialised working directory.
	initStrict initMode = iota
	// initTolerant: a learner running their own commands may be in a situation
	// where the backend has to be initialised again, by them.
	initTolerant
)

// backendProblem says whether a failed init is about the backend.
func backendProblem(out runner.Output) bool {
	text := strings.ToLower(string(out.Stderr) + string(out.Stdout))
	return strings.Contains(text, "backend configuration changed") ||
		strings.Contains(text, "backend initialization required") ||
		strings.Contains(text, "initialization required") ||
		strings.Contains(text, "state migration") ||
		strings.Contains(text, "-migrate-state") ||
		strings.Contains(text, "-reconfigure")
}

// prepareFix makes the run folder ready: it writes the learner's files, builds
// the situation if that has not been done, and runs init.
func (e Exercise) prepareFix(ctx context.Context, env *Env, files map[string][]byte, init initMode) (*run, context.Context, context.CancelFunc, error) {
	if env.Engine == nil {
		return nil, ctx, nil, &SkippedError{fmt.Sprintf("%s needs Terraform: run `tflings doctor`", e.Name)}
	}
	if len(files) == 0 {
		return nil, ctx, nil, &FailedError{Lines: []string{"There are no Terraform files in " + e.Path + "."}}
	}
	if problems := runner.CheckProviders(files); len(problems) > 0 {
		var ls []string
		for _, p := range problems {
			ls = append(ls, p.String())
		}
		return nil, ctx, nil, &FailedError{Lines: append([]string{"tflings will not run this configuration:"}, ls...)}
	}
	steps, err := e.LoadSteps(env.Embedded)
	if err != nil {
		return nil, ctx, nil, err
	}
	r, err := env.newRun(e, files)
	if err != nil {
		return nil, ctx, nil, err
	}
	if err := preflightHosts(ctx, r.cfg.Account, files); err != nil {
		return nil, ctx, nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, e.RunTimeout())
	if _, err := os.Stat(filepath.Join(r.dir, seededMarker)); err != nil {
		if err := r.seed(ctx, steps, files); err != nil {
			cancel()
			return nil, ctx, nil, err
		}
	}
	// Init is run for the learner every time: the lock file is rewritten with the
	// learner's files, and the installed providers have to be checked against it
	// again. When the exercise is about the backend itself, init says that the
	// backend changed and refuses; for `tflings tf` that is the situation to
	// repair, not an error to stop at, so it is let through.
	out, err := r.tf(ctx, "init", nil, nil)
	if err != nil {
		cancel()
		return nil, ctx, nil, err
	}
	if out.ExitCode != 0 && (init != initTolerant || !backendProblem(out)) {
		cancel()
		return nil, ctx, nil, terraformFailed("terraform init could not finish:", out)
	}
	return r, ctx, cancel, nil
}

// seed builds the situation. Failing here is tflings's problem, not the
// learner's, and the message says so.
func (r *run) seed(ctx context.Context, steps SetupSteps, learnerFiles map[string][]byte) error {
	report(ctx, "building the exercise")
	fail := func(what string, err error) error {
		_ = os.RemoveAll(r.src)
		_ = os.Remove(filepath.Join(r.dir, seededMarker))
		return fmt.Errorf("could not build %s (%s): %w\n\nIf it keeps failing, run `tflings reset %s`: it builds the exercise again in a fresh account", r.ex.Name, what, err, r.ex.Name)
	}
	for i, st := range steps.Steps {
		switch {
		case st.Apply != "":
			files, err := rawFiles(r.env.Embedded, path.Join(r.ex.SetupPath(), st.Apply))
			if err != nil {
				return fail("reading setup", err)
			}
			if err := writeSource(r.src, files, true); err != nil {
				return fail("writing setup", err)
			}
			for _, step := range []struct {
				cmd   string
				flags []string
			}{{"init", nil}, {"apply", []string{"-auto-approve"}}} {
				out, err := r.tf(ctx, step.cmd, step.flags, nil)
				if err != nil {
					return fail(fmt.Sprintf("step %d, terraform %s", i+1, step.cmd), err)
				}
				if out.ExitCode != 0 {
					return fail(fmt.Sprintf("step %d, terraform %s", i+1, step.cmd), terraformFailed("Terraform said:", out))
				}
			}
		case st.Mutate != nil:
			client := awscheck.New(runner.Endpoint(), r.cfg.Account)
			if err := client.Mutate(ctx, st.Mutate.Call, st.Mutate.Args); err != nil {
				return fail(fmt.Sprintf("step %d, %s", i+1, st.Mutate.Call), err)
			}
		}
	}
	// What the state looked like when the learner got it.
	if out, err := r.tf(ctx, "show", []string{"-json"}, nil); err == nil && out.ExitCode == 0 {
		_ = os.WriteFile(filepath.Join(r.dir, seedStateFile), out.Stdout, 0o644)
	}
	// Put the learner's files back in place of the setup's.
	if err := writeSource(r.src, learnerFiles, true); err != nil {
		return fail("writing your files", err)
	}
	// The marker also says whether the situation lives in Floci, so that
	// HasSavedState knows that emptying Floci would destroy it.
	marker := "ok\n"
	if r.cfg.Account != "" {
		marker = seededInFloci + "\n"
	}
	return os.WriteFile(filepath.Join(r.dir, seededMarker), []byte(marker), 0o644)
}

func (r *run) seedState() map[string]any {
	data, err := os.ReadFile(filepath.Join(r.dir, seedStateFile))
	if err != nil {
		return nil
	}
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	return m
}

// runFix checks the end state of a fix exercise. It never applies anything:
// repairing is the learner's job. It reads the state, the real thing in Floci,
// and wants a plan with nothing left to do.
func (e Exercise) runFix(ctx context.Context, env *Env, c *check.Compiled, files map[string][]byte) (Result, error) {
	r, ctx, cancel, err := e.prepareFix(ctx, env, files, initStrict)
	if err != nil {
		return Result{}, err
	}
	defer cancel()

	plan, changed, err := r.planFor(ctx, nil, true, true)
	if err != nil {
		return Result{}, err
	}
	show, err := r.tf(ctx, "show", []string{"-json"}, nil)
	if err != nil {
		return Result{}, err
	}
	if show.ExitCode != 0 {
		return Result{}, terraformFailed("terraform show could not read the state:", show)
	}
	var state map[string]any
	if err := json.Unmarshal(show.Stdout, &state); err != nil {
		return Result{}, fmt.Errorf("%s: the state is not valid JSON: %w", e.Name, err)
	}
	failures := c.RunPlanOnly(plan)
	failures = append(failures, c.RunConfigOnly(plan)...)
	failures = append(failures, c.RunFiles(files)...)
	failures = append(failures, c.RunStateSeeded(state, r.seedState())...)
	if len(c.AWS) > 0 {
		awsFailures, err := c.RunAWS(ctx, awscheck.New(runner.Endpoint(), r.cfg.Account), state)
		if err != nil {
			return Result{}, err
		}
		failures = append(failures, awsFailures...)
	}
	if len(failures) > 0 {
		return Result{}, &FailedError{Lines: failures}
	}
	if c.EmptyPlanAfter && changed {
		return Result{}, &FailedError{
			Lines:   []string{"`terraform plan` still wants to change something: the situation is not repaired yet."},
			Details: pendingChanges(plan),
		}
	}
	return Result{Exercise: e, Checked: c.Count(), Explanation: c.Explanation}, nil
}

// Terraform runs one Terraform command of the learner's in the sealed
// environment of the exercise, with their terminal attached. It builds the
// situation first if that has not been done, and puts the learner's current
// files into the run folder. It returns Terraform's exit code.
func (e Exercise) Terraform(ctx context.Context, env *Env, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if e.Mode != ModeFix {
		return -1, fmt.Errorf("%s is a %s exercise: `tflings tf` is for fix exercises, which you repair with Terraform commands", e.Name, e.Mode)
	}
	if reason := e.SkipReason(env.Availability(ctx)); reason != "" {
		return -1, &SkippedError{fmt.Sprintf("%s %s", e.Name, reason)}
	}
	files, err := WorkspaceFiles(os.DirFS(env.workspace()), filepath.ToSlash(e.Path))
	if err != nil {
		return -1, fmt.Errorf("cannot read %s: %w", e.Path, err)
	}
	return e.terraformWith(ctx, env, files, args, stdin, stdout, stderr)
}

// terraformWith is Terraform with the learner's files given: the one path every
// command takes, `tflings tf` and the proof of a solution alike.
func (e Exercise) terraformWith(ctx context.Context, env *Env, files map[string][]byte, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	r, _, cancel, err := e.prepareFix(ctx, env, files, initTolerant)
	if err != nil {
		return -1, err
	}
	cancel() // the run's timeout is for the checks; a learner may take their time
	return runner.Attach(ctx, runner.Call{Engine: env.Engine, Dir: r.src, Env: runner.Env(r.cfg), Args: args}, stdin, stdout, stderr)
}

// SealedEnv is the environment Terraform sees in this exercise, as sorted
// KEY=value lines, for `tflings env`.
func (e Exercise) SealedEnv(env *Env) ([]string, string, error) {
	r, err := env.runFor(e)
	if err != nil {
		return nil, "", err
	}
	out := runner.Env(r.cfg)
	sort.Strings(out)
	return out, r.src, nil
}

// RunSolution plays the solution of a fix exercise: the files it replaces, then
// each command of solution.sh, then the check. The tests use it to prove that
// the situation, as seeded, fails the check and the solution passes it.
func (e Exercise) RunSolution(ctx context.Context, env *Env) (Result, error) {
	files, err := e.ProofFiles(env)
	if err != nil {
		return Result{}, err
	}
	script, err := fs.ReadFile(env.Solutions, e.SolutionPath())
	if err != nil {
		return Result{}, fmt.Errorf("no solution script %s: %w", e.SolutionPath(), err)
	}
	raw, err := fs.ReadFile(env.Embedded, e.CheckPath())
	if err != nil {
		return Result{}, err
	}
	c, err := check.Load(raw)
	if err != nil {
		return Result{}, err
	}
	for _, line := range strings.Split(string(script), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		args, err := splitCommand(line)
		if err != nil || len(args) < 2 || args[0] != "terraform" {
			return Result{}, fmt.Errorf("%s: %q is not a `terraform ...` command", e.SolutionPath(), line)
		}
		// The same path as `tflings tf`: the files are written and init is run
		// before every command, so a file one command leaves for the next
		// (a saved plan) has to survive that.
		var buf bytes.Buffer
		ctx2, cancel := context.WithTimeout(ctx, e.RunTimeout())
		code, err := e.terraformWith(ctx2, env, files, args[1:], strings.NewReader(""), &buf, &buf)
		cancel()
		if err != nil {
			return Result{}, err
		}
		if code != 0 {
			return Result{}, terraformFailed(fmt.Sprintf("solution command failed: %s", line), runner.Output{ExitCode: code, Stdout: buf.Bytes()})
		}
	}
	return e.runFix(ctx, env, c, files)
}

// splitCommand splits a line into words, honouring double and single quotes.
// It never runs a shell: there is no expansion, pipe or redirection.
func splitCommand(line string) ([]string, error) {
	var args []string
	var cur strings.Builder
	var quote rune
	started := false
	for _, ch := range line {
		switch {
		case quote != 0:
			if ch == quote {
				quote = 0
			} else {
				cur.WriteRune(ch)
			}
		case ch == '"' || ch == '\'':
			quote, started = ch, true
		case ch == ' ' || ch == '\t':
			if started || cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in %q", line)
	}
	if started || cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args, nil
}
