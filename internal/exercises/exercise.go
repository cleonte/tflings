// Package exercises loads the manifest (info.toml), tracks which exercises
// are done and runs them.
package exercises

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Modes an exercise can run in. plan only plans, apply really builds in Floci,
// fix repairs a seeded broken situation with the CLI, cli has the learner write
// one Terraform command, quiz is a multiple choice question.
const (
	ModePlan  = "plan"
	ModeApply = "apply"
	ModeFix   = "fix"
	ModeCLI   = "cli"
	ModeQuiz  = "quiz"
)

// File names inside an exercise folder.
const (
	// ExerciseFile is what the learner edits in plan and apply mode.
	ExerciseFile = "main.tf"
	CheckFile    = "check.yaml"
	// SetupDir holds what a fix exercise seeds (Floci state, a tfstate, a lock).
	// Like check.yaml it stays in the binary: it would give the cause away.
	SetupDir   = "setup"
	ReadmeFile = "README.md"
	// ScriptFile is the solution of a fix exercise.
	ScriptFile = "solution.sh"
	// AnswerFile is what the learner writes in a cli exercise, and the reference
	// answer in solutions/. tflings reads it; it never runs it with a shell.
	AnswerFile = "answer.sh"
	// QuizFile is what the learner edits in a quiz exercise.
	QuizFile = "quiz.toml"
)

// Requirements an exercise can name in `requires`.
const (
	RequiresFloci = "floci" // needs the tflings Floci container
	RequiresHCP   = "hcp"   // needs an HCP Terraform token
)

var notDoneRegex = regexp.MustCompile(`(?m)^\s*#\s*I\s+AM\s+NOT\s+DONE`)

// Exercise is one entry of info.toml.
type Exercise struct {
	Name  string
	Path  string // folder, with forward slashes, relative to the workspace
	Topic string
	Mode  string
	Hint  string
	Docs  string
	// Objective is the id on the exam content list, e.g. "4e".
	Objective string
	// Engines lists the engines the exercise works with. Always ["terraform"]
	// until OpenTofu support (Phase 8); empty means terraform.
	Engines []string
	// Requires lists what the exercise needs besides Terraform: "floci", "hcp".
	// apply and fix exercises always need Floci, so they need not say so.
	Requires []string
	// Timeout is how long one run may take, e.g. "60s". Default per mode.
	Timeout string
}

// Errors that commands turn into messages for the learner.
var (
	ErrExerciseNotFound = errors.New("exercise not found")
	ErrNotMarked        = errors.New("the exercise has no 'I AM NOT DONE' marker")
	ErrNoPending        = errors.New("no pending exercises: you are done")
)

var objectiveRe = regexp.MustCompile(`^[1-8][a-h]?$`)

// ParseInfo parses and validates the content of an info.toml.
func ParseInfo(data []byte) ([]Exercise, error) {
	var info struct{ Exercises []Exercise }
	if err := toml.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("info.toml: %w", err)
	}
	seen := map[string]bool{}
	for _, e := range info.Exercises {
		switch {
		case e.Name == "":
			return nil, errors.New("info.toml: an exercise has no name")
		case seen[e.Name]:
			return nil, fmt.Errorf("info.toml: duplicate exercise %q", e.Name)
		case e.Path == "" || e.Topic == "" || e.Hint == "":
			return nil, fmt.Errorf("info.toml: exercise %q needs path, topic and hint", e.Name)
		case !slices.Contains([]string{ModePlan, ModeApply, ModeFix, ModeCLI, ModeQuiz}, e.Mode):
			return nil, fmt.Errorf("info.toml: exercise %q has unknown mode %q", e.Name, e.Mode)
		case e.Objective != "" && !objectiveRe.MatchString(e.Objective):
			return nil, fmt.Errorf("info.toml: exercise %q has objective %q, which is not an exam objective like \"4e\"", e.Name, e.Objective)
		}
		for _, r := range e.Requires {
			if r != RequiresFloci && r != RequiresHCP {
				return nil, fmt.Errorf("info.toml: exercise %q requires unknown %q", e.Name, r)
			}
		}
		for _, en := range e.Engines {
			if en != "terraform" {
				return nil, fmt.Errorf("info.toml: exercise %q names engine %q; only terraform is supported yet", e.Name, en)
			}
		}
		if e.Timeout != "" {
			if _, err := time.ParseDuration(e.Timeout); err != nil {
				return nil, fmt.Errorf("info.toml: exercise %q has timeout %q: %w", e.Name, e.Timeout, err)
			}
		}
		seen[e.Name] = true
	}
	return info.Exercises, nil
}

// List reads the manifest file.
func List(infoFile string) ([]Exercise, error) {
	data, err := os.ReadFile(infoFile)
	if err != nil {
		return nil, err
	}
	return ParseInfo(data)
}

// Find returns the exercise called name.
func Find(name, infoFile string) (Exercise, error) {
	all, err := List(infoFile)
	if err != nil {
		return Exercise{}, err
	}
	for _, e := range all {
		if e.Name == name {
			return e, nil
		}
	}
	return Exercise{}, ErrExerciseNotFound
}

// NextPending returns the first exercise, in manifest order, that is not done
// and can run here.
func NextPending(infoFile string, av Availability) (Exercise, error) {
	all, err := List(infoFile)
	if err != nil {
		return Exercise{}, err
	}
	for _, e := range all {
		if e.State() == Pending && e.SkipReason(av) == "" {
			return e, nil
		}
	}
	return Exercise{}, ErrNoPending
}

// Default timeouts per mode: plan is a quick local run, apply and fix talk to
// Floci and may create several resources.
var defaultTimeouts = map[string]time.Duration{
	ModePlan:  60 * time.Second,
	ModeApply: 180 * time.Second,
	ModeFix:   180 * time.Second,
	ModeCLI:   30 * time.Second,
	ModeQuiz:  5 * time.Second,
}

// RunTimeout is how long a run of the exercise may take.
func (e Exercise) RunTimeout() time.Duration {
	if e.Timeout != "" {
		if d, err := time.ParseDuration(e.Timeout); err == nil {
			return d
		}
	}
	return defaultTimeouts[e.Mode]
}

// NeedsFloci says whether the exercise talks to Floci.
func (e Exercise) NeedsFloci() bool {
	return e.Mode == ModeApply || slices.Contains(e.Requires, RequiresFloci)
}

// NeedsTerraform says whether the exercise runs the terraform binary.
func (e Exercise) NeedsTerraform() bool { return e.Mode != ModeQuiz && e.Mode != ModeCLI }

var todoLineRe = regexp.MustCompile(`#\s*TODO:`)

// Location is File with the line of the first `# TODO:` comment added
// ("exercises/variables/variables1/main.tf:7"), which the terminal of an editor
// such as VS Code turns into a link that opens the file at the fix site. Without
// a file, or a TODO in it, it is File.
func (e Exercise) Location() string {
	file := e.File()
	if e.Mode == ModeFix {
		return file
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return file
	}
	for i, line := range strings.Split(string(data), "\n") {
		if todoLineRe.MatchString(line) {
			return fmt.Sprintf("%s:%d", file, i+1)
		}
	}
	return file
}

// File is the path, on disk, of the file the learner edits. A fix exercise has
// no such file: the learner works with the CLI, and the file that matters is the
// README.md with the ticket.
func (e Exercise) File() string {
	if e.Mode == ModeFix {
		return filepath.FromSlash(path.Join(e.Path, ReadmeFile))
	}
	return filepath.FromSlash(e.Source())
}

// EmbeddedOnly says whether a file of an exercise folder stays in the binary and
// is never written to the learner's workspace: the checks and the setup folder,
// because they would give the cause away. Init, reset and doctor all decide with
// this one function.
func EmbeddedOnly(name string) bool {
	if path.Base(name) == CheckFile {
		return true
	}
	// exercises/<topic>/<name>/setup/...: a topic or exercise called "setup"
	// does not count, only the folder inside an exercise.
	parts := strings.Split(path.Dir(name), "/")
	return len(parts) > 3 && parts[3] == SetupDir
}

// CheckPath is the check.yaml of the exercise, inside the embedded exercise tree
// (always forward slashes).
func (e Exercise) CheckPath() string { return path.Join(e.Path, CheckFile) }

// SetupPath is the setup folder of a fix exercise, inside the embedded tree.
func (e Exercise) SetupPath() string { return path.Join(e.Path, SetupDir) }

// Source is the file the learner edits, inside the embedded tree.
func (e Exercise) Source() string {
	switch e.Mode {
	case ModeCLI:
		return path.Join(e.Path, AnswerFile)
	case ModeQuiz:
		return path.Join(e.Path, QuizFile)
	}
	return path.Join(e.Path, ExerciseFile)
}

// SolutionPath is where the solution lives in the embedded solutions tree.
func (e Exercise) SolutionPath() string {
	dir := path.Join("solutions", strings.TrimPrefix(e.Path, "exercises/"))
	switch e.Mode {
	case ModeFix:
		return path.Join(dir, ScriptFile)
	case ModeCLI:
		return path.Join(dir, AnswerFile)
	case ModeQuiz:
		return path.Join(dir, QuizFile)
	}
	return path.Join(dir, ExerciseFile)
}

// State says whether an exercise is done.
type State int

// The states of an exercise.
const (
	Pending State = iota + 1
	Done
)

func (s State) String() string { return [...]string{"Pending", "Done"}[s-1] }

// State is Done when the check has passed (a fix exercise), or has passed and
// the learner has removed the marker (the others). Taking the marker out of a
// file that does not pass yet does not skip the exercise.
func (e Exercise) State() State {
	if !passed(e.Name) {
		return Pending
	}
	if e.Mode == ModeFix {
		return Done
	}
	if e.MarkerGone() {
		return Done
	}
	return Pending
}

// MarkerGone reports whether the exercise file exists without the marker line.
func (e Exercise) MarkerGone() bool {
	if e.Mode == ModeFix {
		return false
	}
	data, err := os.ReadFile(e.File())
	return err == nil && !notDoneRegex.Match(data)
}

// MarkDone removes the marker line, which is what a learner does by hand.
func (e Exercise) MarkDone() error {
	if e.Mode == ModeFix {
		return nil // a fix exercise is done when its check passes, which records it
	}
	info, err := os.Stat(e.File())
	if err != nil {
		return err
	}
	data, err := os.ReadFile(e.File())
	if err != nil {
		return err
	}
	lines := strings.SplitAfter(string(data), "\n")
	kept := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if notDoneRegex.MatchString(line) {
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	if !removed {
		return ErrNotMarked
	}
	return os.WriteFile(e.File(), []byte(strings.Join(kept, "")), info.Mode().Perm())
}
