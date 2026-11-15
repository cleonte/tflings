// Package exam is the mock exam: a bank of multiple choice questions, tagged by
// exam objective, asked in a terminal and scored per objective.
package exam

import (
	"fmt"
	"io/fs"
	"math/rand"
	"path"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Question is one entry of a file in questions/. Correct holds 0-based indexes
// into Options, as written; the options are shuffled when asked.
type Question struct {
	ID          string   `toml:"id"`
	Objective   string   `toml:"objective"`
	Text        string   `toml:"text"`
	Options     []string `toml:"options"`
	Correct     []int    `toml:"correct"`
	Explanation string   `toml:"explanation"`
}

// Multi says whether the question wants more than one answer ("pick two").
func (q Question) Multi() bool { return len(q.Correct) > 1 }

type file struct {
	Question []Question `toml:"question"`
}

// Load reads every .toml file of dir in fsys.
func Load(fsys fs.FS, dir string) ([]Question, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	var all []Question
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var f file
		if err := toml.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		all = append(all, f.Question...)
	}
	return all, nil
}

// Validate reports what is wrong with a question, or "".
func (q Question) Validate() string {
	switch {
	case q.ID == "":
		return "no id"
	case strings.TrimSpace(q.Text) == "":
		return "no text"
	case len(q.Options) < 2 || len(q.Options) > 5:
		return "needs 2 to 5 options"
	case len(q.Correct) == 0:
		return "no correct answer"
	case strings.TrimSpace(q.Explanation) == "":
		return "no explanation"
	}
	seen := map[int]bool{}
	for _, c := range q.Correct {
		if c < 0 || c >= len(q.Options) {
			return fmt.Sprintf("correct index %d is out of range", c)
		}
		if seen[c] {
			return fmt.Sprintf("correct index %d twice", c)
		}
		seen[c] = true
	}
	if len(q.Correct) >= len(q.Options) {
		return "every option is correct"
	}
	for _, o := range q.Options {
		if len(o) > 1 && o[1] == ')' || strings.TrimSpace(o) == "" {
			return "options must not carry their own letter and must not be empty"
		}
	}
	return ""
}

// Filter keeps the questions of an objective: "4g" is one sub-objective, "4" a
// whole objective. An empty filter keeps all.
func Filter(qs []Question, objective string) []Question {
	if objective == "" {
		return qs
	}
	var out []Question
	for _, q := range qs {
		if q.Objective == objective || (len(objective) == 1 && strings.HasPrefix(q.Objective, objective)) {
			out = append(out, q)
		}
	}
	return out
}

// Pick shuffles the questions with the seed and keeps at most n of them.
func Pick(qs []Question, n int, seed int64) []Question {
	out := slices.Clone(qs)
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// Asked is a question with its options in the order they are shown.
type Asked struct {
	Question
	Shown   []string // options as shown
	Right   []int    // indexes into Shown
	Given   []int    // what the learner answered (nil: skipped)
	Correct bool
	Seen    bool // shown to the learner
}

// Shuffle puts the options of q in a random order.
func Shuffle(q Question, r *rand.Rand) Asked {
	idx := r.Perm(len(q.Options))
	a := Asked{Question: q}
	for pos, orig := range idx {
		a.Shown = append(a.Shown, q.Options[orig])
		if slices.Contains(q.Correct, orig) {
			a.Right = append(a.Right, pos)
		}
	}
	slices.Sort(a.Right)
	return a
}

// ParseAnswer reads "b" or "a,c" or "a c" or "ac" into sorted indexes below max.
func ParseAnswer(s string, max int) ([]int, error) {
	var got []int
	for _, r := range strings.ToLower(s) {
		if r == ',' || r == ' ' {
			continue
		}
		i := int(r - 'a')
		if r < 'a' || i >= max {
			return nil, fmt.Errorf("%q is not one of the options", string(r))
		}
		if !slices.Contains(got, i) {
			got = append(got, i)
		}
	}
	if len(got) == 0 {
		return nil, fmt.Errorf("no answer")
	}
	slices.Sort(got)
	return got, nil
}
