package tui

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/cleonte/tflings/internal/workspace"
)

// session is a workspace on disk and a Model driven by messages, with no
// terminal: the commands the model returns are run here, one after the other.
type session struct {
	t *testing.T
	m *Model
}

func newSession(t *testing.T) *session {
	t.Helper()
	dir := t.TempDir()
	if err := workspace.Init(tflings.FS, filepath.Join(dir, "w")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(dir, "w"))
	m := New(Config{
		InfoFile:  "info.toml",
		Env:       &exercises.Env{Embedded: tflings.FS, Solutions: tflings.Solutions},
		Source:    tflings.FS,
		Solutions: tflings.Solutions,
		Root:      ".",
		Editor:    "",
		NoPoll:    true,
	})
	s := &session{t: t, m: m}
	s.send(startMsg{})
	return s
}

// send delivers a message and runs every command that comes back.
func (s *session) send(msg tea.Msg) {
	s.t.Helper()
	_, cmd := s.m.Update(msg)
	s.run(cmd)
}

func (s *session) run(cmd tea.Cmd) {
	s.t.Helper()
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range msg {
			s.run(c)
		}
	case tea.QuitMsg:
	default:
		_, next := s.m.Update(msg)
		s.run(next)
	}
}

func (s *session) key(k string) {
	s.t.Helper()
	var msg tea.KeyPressMsg
	switch k {
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		msg = tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
	}
	s.send(msg)
}

func (s *session) screen() string {
	return stripANSI(s.m.View().Content)
}

func stripANSI(in string) string {
	var out strings.Builder
	esc := false
	for _, r := range in {
		switch {
		case r == 0x1b:
			esc = true
		case esc && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			esc = false
		case !esc:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func (s *session) expect(want ...string) {
	s.t.Helper()
	screen := s.screen()
	for _, w := range want {
		if !strings.Contains(screen, w) {
			s.t.Errorf("the screen should show %q:\n%s", w, screen)
		}
	}
}

// Not anchored at the end of the line: on Windows git checks files out with
// CRLF line endings.
var answerLine = regexp.MustCompile(`(?m)^answer = "[^"]*"`)

func (s *session) quiz(answer string) {
	s.t.Helper()
	p := "exercises/iac/iac1/quiz.toml"
	data, err := os.ReadFile(p)
	if err != nil {
		s.t.Fatal(err)
	}
	out := answerLine.ReplaceAllString(string(data), `answer = "`+answer+`"`)
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		s.t.Fatal(err)
	}
	s.send(changesMsg{files: []string{p}})
}

func TestStartsOnTheFirstPendingExercise(t *testing.T) {
	s := newSession(t)
	s.expect("tflings", "0/", "iac1", "not solved yet", "Set answer", "h hint")
}

func TestSavingARightAnswerShowsTheExplanationAndNMovesOn(t *testing.T) {
	s := newSession(t)
	s.quiz("a")
	s.expect("not solved yet", "not the right answer")
	s.quiz("b")
	s.expect("iac1 works!", "describing infrastructure in files", "I AM NOT DONE")
	// n takes out the marker, which is what the learner would do by hand.
	s.key("n")
	data, _ := os.ReadFile("exercises/iac/iac1/quiz.toml")
	if strings.Contains(string(data), "I AM NOT DONE") {
		t.Error("n should remove the marker")
	}
	if got := s.m.notice; !strings.Contains(got, "iac1 is done") {
		t.Errorf("notice %q", got)
	}
}

func TestNDoesNotSkipAnExerciseThatDoesNotWork(t *testing.T) {
	s := newSession(t)
	s.key("n")
	s.expect("Get iac1 to work first")
	data, _ := os.ReadFile("exercises/iac/iac1/quiz.toml")
	if !strings.Contains(string(data), "I AM NOT DONE") {
		t.Error("the marker must stay")
	}
}

func TestHintAndEscape(t *testing.T) {
	s := newSession(t)
	s.key("h")
	s.expect("Hint for iac1", "description of the infrastructure", "Docs:")
	s.key("esc")
	s.expect("not solved yet")
}

func TestSolutionAsksFirst(t *testing.T) {
	s := newSession(t)
	s.key("s")
	s.expect("Show the solution of iac1?", "y to show it")
	s.key("x") // any other key goes back
	s.expect("not solved yet")
	s.key("s")
	s.key("y")
	s.expect("Solution for iac1", `answer = "b"`)
}

func TestResetBringsBackTheFileAndAFreshAccount(t *testing.T) {
	s := newSession(t)
	s.quiz("b")
	s.expect("works!")
	if exercises.ResetCount(".", "iac1") != 0 {
		t.Fatal("not reset yet")
	}
	s.key("r")
	s.expect("Reset iac1 to its original state?", "fresh AWS account")
	s.key("y")
	data, _ := os.ReadFile("exercises/iac/iac1/quiz.toml")
	if !strings.Contains(string(data), `answer = ""`) || !strings.Contains(string(data), "I AM NOT DONE") {
		t.Errorf("the original file should be back:\n%s", data)
	}
	if exercises.ResetCount(".", "iac1") != 1 {
		t.Error("the reset must be counted, so that the exercise gets a fresh account")
	}
	s.expect("not solved yet")
}

func TestEverythingDone(t *testing.T) {
	s := newSession(t)
	all, _ := exercises.List("info.toml")
	for _, ex := range all {
		if ex.Mode != exercises.ModeQuiz && ex.Mode != exercises.ModeCLI {
			continue
		}
		if err := exercises.RecordPass(ex.Name); err != nil {
			t.Fatal(err)
		}
		_ = ex.MarkDone()
	}
	// Plan exercises need Terraform, which this environment has none of, so they
	// are skipped and the quizzes and cli exercises (which need none) are all that is left.
	s.send(changesMsg{})
	s.expect("Congratulations", "skipped (see tflings doctor)")
}

func TestListJumpsToAnExercise(t *testing.T) {
	s := newSession(t)
	s.key("l")
	s.expect("iac1")
	s.key("esc")
	s.expect("not solved yet")
}

func TestQuitStopsTheRun(t *testing.T) {
	s := newSession(t)
	_, cmd := s.m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("q should produce a quit message")
	}
}

func TestEditNeedsAnEditor(t *testing.T) {
	s := newSession(t)
	s.key("e")
	s.expect("Set $VISUAL or $EDITOR")
}

func TestEditorCommand(t *testing.T) {
	if editorCommand("", "f") != nil {
		t.Error("no editor, no command")
	}
	c := editorCommand("code -w", "main.tf")
	if c == nil || c.Args[0] != "code" || c.Args[len(c.Args)-1] != "main.tf" || c.Args[1] != "-w" {
		t.Errorf("unexpected command %v", c)
	}
}

func TestRunsReportProgress(t *testing.T) {
	ctx := exercises.WithProgress(context.Background(), func(string) {})
	if ctx == nil {
		t.Fatal("no context")
	}
}

func TestLongLinesAreWrappedNotCut(t *testing.T) {
	m := &Model{width: 40}
	got := m.wrap("This is a long explanation that does not fit in forty columns at all.", 4)
	if len(got) < 2 {
		t.Fatalf("it should wrap onto several lines, got %q", got)
	}
	for _, l := range got {
		if lipgloss.Width(l) > 36 {
			t.Errorf("line %q is wider than the space left", l)
		}
	}
	if joined := strings.Join(strings.Fields(strings.Join(got, " ")), " "); !strings.HasSuffix(joined, "at all.") {
		t.Errorf("no words may be lost: %q", joined)
	}
	if one := (&Model{}).wrap("no width known", 0); len(one) != 1 || one[0] != "no width known" {
		t.Errorf("without a width the text is left alone: %q", one)
	}
}

func TestGChecksTheExerciseAgain(t *testing.T) {
	s := newSession(t)
	s.expect("iac1")
	before := s.m.runID
	s.key("g")
	if s.m.runID == before {
		t.Error("g should start a new run")
	}
	s.expect("iac1")
}

func TestAFixExerciseIsCheckedAgainWhenTfRan(t *testing.T) {
	s := newSession(t)
	s.m.exercise = exercises.Exercise{Name: "backends3", Mode: exercises.ModeFix}
	s.m.status, s.m.mode = statusFinished, modeResult
	env := &exercises.Env{Home: t.TempDir(), Embedded: tflings.FS, Solutions: tflings.Solutions}
	s.m.cfg.Env = env
	s.m.cfg.NoPoll = false

	// Nothing ran: the tick only asks again.
	if _, cmd := s.m.Update(tickMsg{}); cmd == nil {
		t.Error("the tick should be scheduled again")
	}
	if s.m.status != statusFinished {
		t.Errorf("no tf ran, so no new run: status %v", s.m.status)
	}

	// `tflings tf` ended a command.
	if err := env.MarkTfRan("backends3"); err != nil {
		t.Fatal(err)
	}
	s.m.Update(tickMsg{})
	if s.m.status != statusRunning {
		t.Errorf("tf ran, so the exercise should be checked again: status %v", s.m.status)
	}
}
