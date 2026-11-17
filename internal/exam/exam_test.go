package exam_test

import (
	"bytes"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/exam"
)

var objectiveID = regexp.MustCompile(`^[1-8][a-h]$`)

// The shipped bank: every question is well formed, ids are unique, and the
// objective is one of the content list (the sub-objective letters of the
// official list are checked in internal/exercises).
func TestShippedQuestions(t *testing.T) {
	qs, err := exam.Load(tflings.Questions, "questions")
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) < 30 {
		t.Errorf("the bank has %d questions, want at least 30", len(qs))
	}
	seen := map[string]bool{}
	for _, q := range qs {
		if seen[q.ID] {
			t.Errorf("%s: duplicate id", q.ID)
		}
		seen[q.ID] = true
		if msg := q.Validate(); msg != "" {
			t.Errorf("%s: %s", q.ID, msg)
		}
		if !objectiveID.MatchString(q.Objective) {
			t.Errorf("%s: objective %q is not like 4g", q.ID, q.Objective)
		}
	}
}

func TestFilter(t *testing.T) {
	qs := []exam.Question{{ID: "a", Objective: "4g"}, {ID: "b", Objective: "4h"}, {ID: "c", Objective: "5a"}}
	if got := exam.Filter(qs, "4"); len(got) != 2 {
		t.Errorf("objective 4: got %d", len(got))
	}
	if got := exam.Filter(qs, "4g"); len(got) != 1 || got[0].ID != "a" {
		t.Errorf("objective 4g: got %v", got)
	}
	if got := exam.Filter(qs, ""); len(got) != 3 {
		t.Errorf("no filter: got %d", len(got))
	}
}

func TestShuffleKeepsTheRightAnswers(t *testing.T) {
	q := exam.Question{Options: []string{"w", "x", "y", "z"}, Correct: []int{1, 3}}
	for seed := int64(0); seed < 20; seed++ {
		a := exam.Shuffle(q, rand.New(rand.NewSource(seed)))
		var got []string
		for _, i := range a.Right {
			got = append(got, a.Shown[i])
		}
		if len(got) != 2 || !contains(got, "x") || !contains(got, "z") {
			t.Fatalf("seed %d: right options are %v", seed, got)
		}
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestParseAnswer(t *testing.T) {
	for in, want := range map[string]string{"b": "[1]", "a,c": "[0 2]", "c a": "[0 2]", "ca": "[0 2]", "B": "[1]"} {
		got, err := exam.ParseAnswer(in, 4)
		if err != nil {
			t.Errorf("%q: %v", in, err)
		}
		if sprint(got) != want {
			t.Errorf("%q: got %v, want %s", in, got, want)
		}
	}
	for _, bad := range []string{"", "e", "1", "a;"} {
		if _, err := exam.ParseAnswer(bad, 4); err == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
}

func sprint(v []int) string {
	var b strings.Builder
	b.WriteString("[")
	for i, x := range v {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(string(rune('0' + x)))
	}
	b.WriteString("]")
	return b.String()
}

func tomlBank() []exam.Question {
	fsys := fstest.MapFS{"q/a.toml": {Data: []byte(`
[[question]]
id = "one"
objective = "3b"
text = "Which one?"
options = ["no", "yes"]
correct = [1]
explanation = "Because."
`)}}
	qs, err := exam.Load(fsys, "q")
	if err != nil {
		panic(err)
	}
	return qs
}

func TestSitScoresAndReports(t *testing.T) {
	qs := tomlBank()
	// Find which letter is right for this seed by asking once.
	probe := exam.Shuffle(qs[0], rand.New(rand.NewSource(7)))
	right := string(rune('a' + probe.Right[0]))
	for _, tc := range []struct {
		in    string
		right int
	}{{right + "\n", 1}, {string(rune('a'+1-probe.Right[0])) + "\n", 0}, {"\n", 0}} {
		var out bytes.Buffer
		o := exam.Options{Count: 1, Seed: 7, Exercises: map[string][]string{"3b": {"workflow4"}}}
		res := exam.Sit(strings.NewReader(tc.in), &out, qs, o)
		if res.Right != tc.right {
			t.Errorf("input %q: right = %d, want %d", tc.in, res.Right, tc.right)
		}
		exam.Report(&out, res, o)
		if tc.right == 0 && !strings.Contains(out.String(), "workflow4") {
			t.Errorf("a miss should point to the exercises:\n%s", out.String())
		}
		if !strings.Contains(out.String(), "3b") {
			t.Errorf("the report is by objective:\n%s", out.String())
		}
	}
}

func TestSitRefusesABadAnswerThenAsksAgain(t *testing.T) {
	var out bytes.Buffer
	res := exam.Sit(strings.NewReader("z\n\n"), &out, tomlBank(), exam.Options{Count: 1, Seed: 1})
	if res.Answered != 0 || !strings.Contains(out.String(), "is not one of the options") {
		t.Errorf("answered=%d out=%s", res.Answered, out.String())
	}
}

func TestSitStopsWhenTimeIsUp(t *testing.T) {
	qs := append(tomlBank(), tomlBank()...)
	qs[1].ID = "two"
	now := time.Unix(0, 0)
	clock := func() time.Time { t := now; now = now.Add(40 * time.Minute); return t }
	var out bytes.Buffer
	res := exam.Sit(strings.NewReader("a\na\n"), &out, qs, exam.Options{Count: 2, Seed: 1, Duration: time.Hour, Now: clock})
	if !res.TimedOut {
		t.Errorf("the second question should find the time up: %+v", res)
	}
	if len(res.Asked) != 2 {
		t.Errorf("unasked questions still count in the report, got %d", len(res.Asked))
	}
}

func TestQuitCountsTheRestAsMissed(t *testing.T) {
	qs := append(tomlBank(), tomlBank()...)
	qs[1].ID = "two"
	var out bytes.Buffer
	res := exam.Sit(strings.NewReader("q\n"), &out, qs, exam.Options{Count: 2, Seed: 1})
	if !res.Quit || len(res.Asked) != 2 || res.Right != 0 {
		t.Errorf("%+v", res)
	}
}

func TestSitInsistsOnTheNumberOfAnswers(t *testing.T) {
	qs := []exam.Question{{ID: "m", Objective: "4c", Text: "Pick two", Options: []string{"a", "b", "c"}, Correct: []int{0, 1}, Explanation: "x"}}
	var out bytes.Buffer
	res := exam.Sit(strings.NewReader("a\n\n"), &out, qs, exam.Options{Count: 1, Seed: 1})
	if res.Answered != 0 || !strings.Contains(out.String(), "wants 2 answer") {
		t.Errorf("one answer to a pick-two question must be refused: %s", out.String())
	}
}

func TestHistoryAndWeakest(t *testing.T) {
	dir := t.TempDir()
	if h, err := exam.LoadHistory(dir); err != nil || len(h) != 0 {
		t.Fatalf("no file is an empty history: %v %v", h, err)
	}
	res := exam.Result{Asked: []exam.Asked{
		{Question: exam.Question{Objective: "4g"}, Seen: true, Correct: true},
		{Question: exam.Question{Objective: "4g"}, Seen: true},
		{Question: exam.Question{Objective: "6b"}, Seen: true, Correct: true},
		{Question: exam.Question{Objective: "7c"}}, // never shown: not counted
	}}
	if err := exam.Append(dir, exam.RecordOf(res, false, time.Now())); err != nil {
		t.Fatal(err)
	}
	h, err := exam.LoadHistory(dir)
	if err != nil || len(h) != 1 {
		t.Fatalf("history: %v %v", h, err)
	}
	tot := exam.Totals(h)
	if tot["4g"] != (exam.Tally{Right: 1, N: 2}) || tot["6b"] != (exam.Tally{Right: 1, N: 1}) {
		t.Errorf("totals: %v", tot)
	}
	if _, ok := tot["7c"]; ok {
		t.Error("a question that was not shown must not count")
	}
	qs := []exam.Question{{Objective: "4g"}, {Objective: "6b"}, {Objective: "7c"}, {Objective: "8a"}}
	got := exam.Weakest(h, qs, 3)
	want := []string{"7c", "8a", "4g"} // never asked first, then the lowest score
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("weakest = %v, want %v", got, want)
		}
	}
	if n := len(exam.FilterSet(qs, got)); n != 3 {
		t.Errorf("filter set: %d", n)
	}
}

func TestObjectivesListIsComplete(t *testing.T) {
	if len(exam.Objectives) != 37 {
		t.Errorf("the 004 list has 37 sub-objectives, got %d", len(exam.Objectives))
	}
}

func TestCodeIsShownUnderTheQuestion(t *testing.T) {
	q := exam.Question{ID: "c", Objective: "4e", Text: "What?", Code: "resource \"x\" \"y\" {}\n", Options: []string{"a1", "b1"}, Correct: []int{0}, Explanation: "e"}
	var out bytes.Buffer
	exam.Sit(strings.NewReader("\n"), &out, []exam.Question{q}, exam.Options{Count: 1, Seed: 1})
	if !strings.Contains(out.String(), "What?\n\n    resource \"x\" \"y\" {}\n\n  a)") {
		t.Errorf("the snippet should be indented under the question:\n%s", out.String())
	}
}

func TestReviewUsesTheLastSittingWithMisses(t *testing.T) {
	miss := func(ids ...string) exam.Record {
		return exam.Record{Objectives: map[string]exam.Tally{"4e": {N: 1}}, Missed: ids}
	}
	h := []exam.Record{miss("a", "b"), miss("c"), {Objectives: map[string]exam.Tally{"4e": {Right: 1, N: 1}}}}
	got := exam.LastMissed(h)
	if len(got) != 1 || got[0] != "c" {
		t.Errorf("the latest sitting that missed something: got %v", got)
	}
	if exam.LastMissed(nil) != nil {
		t.Error("no history, nothing to review")
	}
	qs := []exam.Question{{ID: "a"}, {ID: "c"}, {ID: "z"}}
	if f := exam.FilterIDs(qs, got); len(f) != 1 || f[0].ID != "c" {
		t.Errorf("filter ids: %v", f)
	}
	res := exam.Result{Asked: []exam.Asked{
		{Question: exam.Question{ID: "q1", Objective: "4e"}, Seen: true},
		{Question: exam.Question{ID: "q2", Objective: "4e"}, Seen: true, Correct: true},
		{Question: exam.Question{ID: "q3", Objective: "4e"}},
	}}
	if m := exam.RecordOf(res, false, time.Now()).Missed; len(m) != 1 || m[0] != "q1" {
		t.Errorf("only shown and missed questions are recorded: %v", m)
	}
}
