package exam

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"time"
)

// Options of one sitting.
type Options struct {
	Count    int           // questions to ask
	Duration time.Duration // 0: no time limit
	Practice bool          // say right or wrong after each question
	Seed     int64
	Now      func() time.Time
	// Exercises maps an objective to the names of exercises that practise it,
	// for the report.
	Exercises map[string][]string
}

// Result is the outcome of a sitting.
type Result struct {
	Asked     []Asked
	Answered  int
	Right     int
	TimedOut  bool
	Quit      bool
	Remaining time.Duration
}

// Sit asks the questions on out, reading answers from in.
func Sit(in io.Reader, out io.Writer, qs []Question, o Options) Result {
	if o.Now == nil {
		o.Now = time.Now
	}
	r := rand.New(rand.NewSource(o.Seed))
	picked := Pick(qs, o.Count, o.Seed)
	start := o.Now()
	sc := bufio.NewScanner(in)
	var res Result
	for i, q := range picked {
		a := Shuffle(q, r)
		left := time.Duration(0)
		if o.Duration > 0 {
			left = o.Duration - o.Now().Sub(start)
			if left <= 0 {
				res.TimedOut = true
				res.Asked = append(res.Asked, askedRest(picked[i:], r)...)
				break
			}
		}
		head := fmt.Sprintf("\nQuestion %d of %d", i+1, len(picked))
		if o.Duration > 0 {
			head += fmt.Sprintf("   (%s left)", left.Round(time.Second))
		}
		fmt.Fprintln(out, head)
		fmt.Fprintln(out, q.Text)
		for j, opt := range a.Shown {
			fmt.Fprintf(out, "  %c) %s\n", 'a'+j, opt)
		}
		want := "Your answer"
		if q.Multi() {
			want = fmt.Sprintf("Choose %d (for example a,c)", len(q.Correct))
		}
		for {
			fmt.Fprintf(out, "%s, Enter to skip, q to finish: ", want)
			if !sc.Scan() {
				res.Quit = true
				break
			}
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				break
			}
			if strings.EqualFold(line, "q") {
				res.Quit = true
				break
			}
			given, err := ParseAnswer(line, len(a.Shown))
			if err != nil {
				fmt.Fprintf(out, "  %v\n", err)
				continue
			}
			if len(given) != len(q.Correct) {
				fmt.Fprintf(out, "  This question wants %d answer(s).\n", len(q.Correct))
				continue
			}
			a.Given = given
			break
		}
		if res.Quit {
			res.Asked = append(res.Asked, askedRest(picked[i:], r)...)
			break
		}
		if a.Given != nil {
			res.Answered++
			a.Correct = slices.Equal(a.Given, a.Right)
			if a.Correct {
				res.Right++
			}
			if o.Practice {
				feedback(out, a)
			}
		}
		res.Asked = append(res.Asked, a)
	}
	if o.Duration > 0 {
		res.Remaining = max(0, o.Duration-o.Now().Sub(start))
	}
	return res
}

func askedRest(qs []Question, r *rand.Rand) []Asked {
	var out []Asked
	for _, q := range qs {
		out = append(out, Shuffle(q, r))
	}
	return out
}

func letters(idx []int) string {
	var s []string
	for _, i := range idx {
		s = append(s, string(rune('a'+i)))
	}
	return strings.Join(s, ",")
}

func feedback(out io.Writer, a Asked) {
	if a.Correct {
		fmt.Fprintln(out, "  Right.")
	} else {
		fmt.Fprintf(out, "  Wrong. The answer is %s.\n", letters(a.Right))
	}
	fmt.Fprintf(out, "  %s\n", strings.TrimSpace(a.Explanation))
}

// Report prints the score, the score per objective, and what to review.
func Report(out io.Writer, res Result, o Options) {
	total := len(res.Asked)
	fmt.Fprintln(out, "\n── Result ──")
	switch {
	case res.TimedOut:
		fmt.Fprintln(out, "Time is up.")
	case res.Quit:
		fmt.Fprintln(out, "Finished early.")
	}
	if total == 0 {
		return
	}
	fmt.Fprintf(out, "%d of %d right (%d%%); %d answered, %d not answered (they count as wrong).\n",
		res.Right, total, res.Right*100/total, res.Answered, total-res.Answered)
	fmt.Fprintln(out, "HashiCorp does not publish the pass mark or the number of questions of the real exam, so this score is only a guide.")

	type tally struct{ right, n int }
	per := map[string]*tally{}
	for _, a := range res.Asked {
		t := per[a.Objective]
		if t == nil {
			t = &tally{}
			per[a.Objective] = t
		}
		t.n++
		if a.Correct {
			t.right++
		}
	}
	objs := make([]string, 0, len(per))
	for k := range per {
		objs = append(objs, k)
	}
	sort.Strings(objs)
	fmt.Fprintln(out, "\nBy objective:")
	var weak []string
	for _, k := range objs {
		t := per[k]
		fmt.Fprintf(out, "  %-4s %d of %d\n", k, t.right, t.n)
		if t.right < t.n {
			weak = append(weak, k)
		}
	}

	var wrong []Asked
	for _, a := range res.Asked {
		if !a.Correct {
			wrong = append(wrong, a)
		}
	}
	if len(wrong) > 0 && !o.Practice {
		fmt.Fprintln(out, "\nWhat you missed:")
		for _, a := range wrong {
			fmt.Fprintf(out, "\n[%s] %s\n", a.Objective, a.Text)
			if a.Given == nil {
				fmt.Fprintln(out, "  You did not answer.")
			} else {
				fmt.Fprintf(out, "  You answered %s.\n", letters(a.Given))
			}
			var right []string
			for _, i := range a.Right {
				right = append(right, a.Shown[i])
			}
			fmt.Fprintf(out, "  Right: %s\n  %s\n", strings.Join(right, "; "), strings.TrimSpace(a.Explanation))
		}
	}
	var names []string
	for _, k := range weak {
		names = append(names, o.Exercises[k]...)
	}
	slices.Sort(names)
	names = slices.Compact(names)
	if len(names) > 0 {
		fmt.Fprintf(out, "\nExercises for the objectives you missed (%s):\n  %s\n", strings.Join(weak, ", "), strings.Join(names, " "))
	}
}
