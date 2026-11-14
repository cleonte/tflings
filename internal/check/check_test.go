package check

import "testing"

func TestQuizOK(t *testing.T) {
	c, err := Load([]byte("answer: [B]\nexplanation: why\n"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		given []string
		want  bool
	}{
		{[]string{"b"}, true},
		{[]string{" B "}, true},
		{[]string{"a"}, false},
		{[]string{"a", "b"}, false},
		{nil, false},
	}
	for _, tc := range cases {
		if got := c.QuizOK(tc.given); got != tc.want {
			t.Errorf("QuizOK(%v) = %v, want %v", tc.given, got, tc.want)
		}
	}

	multi, _ := Load([]byte("answer: [a, c]\n"))
	if !multi.QuizOK([]string{"c", "a"}) || multi.QuizOK([]string{"a"}) {
		t.Error("a multiple-answer quiz needs exactly the right letters, in any order")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	if _, err := Load([]byte("anwser: [b]\n")); err == nil {
		t.Error("a typo in check.yaml must be an error")
	}
}

func TestRunPlanAndVariants(t *testing.T) {
	c, err := Load([]byte(`
plan:
  - expr: plan.planned_values.outputs.n.value == 3
    message: "n should be 3"
  - expr: plan.planned_values.outputs.missing.value == 1
    message: "missing output"
config:
  - expr: config.root_module.variables.x.default == "a"
    message: "x default"
variants:
  - vars: { x: "b" }
    plan:
      - expr: plan.planned_values.outputs.n.value == 4
        message: "variant n"
`))
	if err != nil {
		t.Fatal(err)
	}
	plan := map[string]any{
		"planned_values": map[string]any{"outputs": map[string]any{"n": map[string]any{"value": float64(3)}}},
		"configuration":  map[string]any{"root_module": map[string]any{"variables": map[string]any{"x": map[string]any{"default": "a"}}}},
	}
	got := c.RunPlan(plan)
	if len(got) != 1 || got[0] != "missing output" {
		t.Errorf("an unevaluable expression must fail with its message, got %v", got)
	}
	if got := c.RunVariant(0, plan); len(got) != 1 || got[0] != "variant n" {
		t.Errorf("variant: got %v", got)
	}
	if c.Count() != 4 {
		t.Errorf("Count = %d, want 4", c.Count())
	}
}

func TestLoadRejectsBadAssertions(t *testing.T) {
	for name, doc := range map[string]string{
		"no message":     "plan:\n  - expr: \"true\"\n",
		"not boolean":    "plan:\n  - expr: \"1 + 1\"\n    message: m\n",
		"syntax error":   "plan:\n  - expr: \"plan.\"\n    message: m\n",
		"variant no var": "variants:\n  - plan:\n      - expr: \"true\"\n        message: m\n",
		"variant empty":  "variants:\n  - vars: { a: b }\n",
	} {
		if _, err := Load([]byte(doc)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestApplyStages(t *testing.T) {
	c, err := Load([]byte(`
plan:
  - expr: plan.resource_changes.size() == 1
    message: "plan"
config:
  - expr: config.root_module.resources.size() == 1
    message: "config"
state:
  - expr: state.values.root_module.resources.exists(r, r.address == "a.b")
    message: "state"
empty_plan_after: true
`))
	if err != nil {
		t.Fatal(err)
	}
	plan := map[string]any{
		"resource_changes": []any{map[string]any{}},
		"configuration":    map[string]any{"root_module": map[string]any{"resources": []any{}}},
	}
	if got := c.RunPlanOnly(plan); len(got) != 0 {
		t.Errorf("plan stage: %v", got)
	}
	if got := c.RunConfigOnly(plan); len(got) != 1 || got[0] != "config" {
		t.Errorf("config stage: %v", got)
	}
	state := map[string]any{"values": map[string]any{"root_module": map[string]any{"resources": []any{map[string]any{"address": "a.b"}}}}}
	if got := c.RunState(state); len(got) != 0 {
		t.Errorf("state stage: %v", got)
	}
	if got := c.RunState(map[string]any{}); len(got) != 1 || got[0] != "state" {
		t.Errorf("a missing state is a failed assertion, got %v", got)
	}
	if c.Count() != 4 {
		t.Errorf("Count = %d, want 4 (plan, config, state, empty plan)", c.Count())
	}
}
