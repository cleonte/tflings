// Package check loads and runs the check.yaml of an exercise. check.yaml is
// embedded in the binary and never written to the learner's workspace.
package check

import (
	"fmt"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"
)

// Check is the content of a check.yaml. Only the quiz part exists so far; the
// plan, config, state and aws parts come with plan mode.
type Check struct {
	// Answer is the right choice of a quiz: one letter, or several for a
	// multiple-answer question. Case does not matter.
	Answer []string `json:"answer,omitempty"`
	// Explanation is shown after a quiz passes.
	Explanation string `json:"explanation,omitempty"`
}

// Load parses a check.yaml.
func Load(raw []byte) (*Check, error) {
	var c Check
	if err := yaml.UnmarshalStrict(raw, &c); err != nil {
		return nil, fmt.Errorf("check.yaml: %w", err)
	}
	for i, a := range c.Answer {
		c.Answer[i] = strings.ToLower(strings.TrimSpace(a))
	}
	return &c, nil
}

// QuizOK reports whether the letters given are exactly the right ones.
func (c *Check) QuizOK(given []string) bool {
	if len(c.Answer) == 0 || len(given) != len(c.Answer) {
		return false
	}
	g := make([]string, len(given))
	for i, a := range given {
		g[i] = strings.ToLower(strings.TrimSpace(a))
	}
	slices.Sort(g)
	want := slices.Clone(c.Answer)
	slices.Sort(want)
	return slices.Equal(g, want)
}
