package exam

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Tally is how many questions of an objective were right out of how many.
type Tally struct {
	Right int `json:"right"`
	N     int `json:"n"`
}

// Record is one finished sitting, as kept in the history file.
type Record struct {
	Time       time.Time        `json:"time"`
	Practice   bool             `json:"practice,omitempty"`
	Objectives map[string]Tally `json:"objectives"`
	// Missed holds the ids of the questions that were shown and answered wrongly
	// (or skipped), for `exam --review`.
	Missed []string `json:"missed,omitempty"`
}

// HistoryFile is the name of the history inside tflings's home folder.
const HistoryFile = "exam-history.jsonl"

// RecordOf summarizes a sitting: only the questions that were shown count.
func RecordOf(res Result, practice bool, now time.Time) Record {
	rec := Record{Time: now.UTC(), Practice: practice, Objectives: map[string]Tally{}}
	for _, a := range res.Asked {
		if !a.Seen {
			continue
		}
		t := rec.Objectives[a.Objective]
		t.N++
		if a.Correct {
			t.Right++
		} else if a.ID != "" {
			rec.Missed = append(rec.Missed, a.ID)
		}
		rec.Objectives[a.Objective] = t
	}
	return rec
}

// Append adds a record to the history in dir.
func Append(dir string, rec Record) error {
	if len(rec.Objectives) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, HistoryFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

// LoadHistory reads the history in dir. A missing file is an empty history, and
// a line that cannot be read is skipped.
func LoadHistory(dir string) ([]Record, error) {
	f, err := os.Open(filepath.Join(dir, HistoryFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Record
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var r Record
		if json.Unmarshal(sc.Bytes(), &r) == nil {
			out = append(out, r)
		}
	}
	return out, sc.Err()
}

// Totals adds up the history per objective.
func Totals(h []Record) map[string]Tally {
	m := map[string]Tally{}
	for _, r := range h {
		for k, t := range r.Objectives {
			c := m[k]
			c.Right += t.Right
			c.N += t.N
			m[k] = c
		}
	}
	return m
}

// Weakest returns the k objectives that have questions and the lowest score,
// those never asked first.
func Weakest(h []Record, qs []Question, k int) []string {
	have := map[string]bool{}
	for _, q := range qs {
		have[q.Objective] = true
	}
	tot := Totals(h)
	var ids []string
	for id := range have {
		ids = append(ids, id)
	}
	score := func(id string) float64 {
		t := tot[id]
		if t.N == 0 {
			return -1
		}
		return float64(t.Right) / float64(t.N)
	}
	sort.Slice(ids, func(i, j int) bool {
		if score(ids[i]) != score(ids[j]) {
			return score(ids[i]) < score(ids[j])
		}
		return ids[i] < ids[j]
	})
	if len(ids) > k {
		ids = ids[:k]
	}
	return ids
}

// FilterSet keeps the questions whose objective is in ids.
func FilterSet(qs []Question, ids []string) []Question {
	var out []Question
	for _, q := range qs {
		for _, id := range ids {
			if q.Objective == id {
				out = append(out, q)
				break
			}
		}
	}
	return out
}

// LastMissed returns the ids of the questions missed in the latest sitting that
// missed any.
func LastMissed(h []Record) []string {
	for i := len(h) - 1; i >= 0; i-- {
		if len(h[i].Missed) > 0 {
			return h[i].Missed
		}
	}
	return nil
}

// FilterIDs keeps the questions with one of the ids.
func FilterIDs(qs []Question, ids []string) []Question {
	var out []Question
	for _, q := range qs {
		for _, id := range ids {
			if q.ID == id {
				out = append(out, q)
				break
			}
		}
	}
	return out
}
