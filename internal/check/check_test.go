package check

import (
	"context"
	"testing"
)

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

type fakeAWS struct{ calls int }

func (f *fakeAWS) Call(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	f.calls++
	return map[string]any{"status": "Enabled", "exists": true}, nil
}

func TestRunAWS(t *testing.T) {
	c, err := Load([]byte(`
aws:
  - call: s3.versioning
    args: { bucket: b }
    expr: result.status == "Enabled"
    message: "enabled"
  - call: s3.versioning
    args: { bucket: b }
    expr: result.status == "Suspended"
    message: "suspended"
`))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeAWS{}
	got, err := c.RunAWS(context.Background(), f, nil)
	if err != nil || len(got) != 1 || got[0] != "suspended" {
		t.Errorf("got %v, %v", got, err)
	}
	if f.calls != 1 {
		t.Errorf("equal calls are made once, got %d", f.calls)
	}
	if c.Count() != 2 {
		t.Errorf("Count = %d", c.Count())
	}
}

func TestLoadRejectsBadAWSChecks(t *testing.T) {
	for name, doc := range map[string]string{
		"unknown call":   "aws:\n  - call: s3.delete_bucket\n    args: {bucket: b}\n    expr: \"true\"\n    message: m\n",
		"missing arg":    "aws:\n  - call: s3.versioning\n    expr: \"true\"\n    message: m\n",
		"no message":     "aws:\n  - call: s3.bucket\n    args: {bucket: b}\n    expr: result.exists\n",
		"bad expression": "aws:\n  - call: s3.bucket\n    args: {bucket: b}\n    expr: \"result.\"\n    message: m\n",
		"unknown field":  "aws:\n  - call: s3.bucket\n    args: {bucket: b}\n    expr: result.exists\n    message: m\n    extra: 1\n",
	} {
		if _, err := Load([]byte(doc)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestVariantFeatures(t *testing.T) {
	c, err := Load([]byte(`
variants:
  - vars: { v: "2" }
    apply_first: { v: "1" }
    plan:
      - expr: plan.resource_changes.size() == 1
        message: "m"
  - vars: { v: "3" }
    apply_first: { v: "1" }
    expect_error: "prevent_destroy"
    message: "refused"
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Variants[0].ApplyFirst["v"] != "1" || c.Variants[1].ExpectError != "prevent_destroy" {
		t.Errorf("unexpected variants %+v", c.Variants)
	}
	if c.Count() != 2 {
		t.Errorf("Count = %d, want 2 (one assertion, one expected error)", c.Count())
	}
	for name, doc := range map[string]string{
		"expect_error without message": "variants:\n  - vars: {v: \"1\"}\n    expect_error: x\n",
		"expect_error with assertions": "variants:\n  - vars: {v: \"1\"}\n    expect_error: x\n    message: m\n    plan:\n      - expr: \"true\"\n        message: m\n",
	} {
		if _, err := Load([]byte(doc)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSeedVariable(t *testing.T) {
	c, err := Load([]byte(`
state:
  - expr: state.id == seed.id
    message: "the object was rebuilt"
`))
	if err != nil {
		t.Fatal(err)
	}
	if f := c.RunStateSeeded(map[string]any{"id": "a"}, map[string]any{"id": "a"}); len(f) != 0 {
		t.Errorf("same id should pass: %v", f)
	}
	if f := c.RunStateSeeded(map[string]any{"id": "b"}, map[string]any{"id": "a"}); len(f) != 1 {
		t.Errorf("a different id should fail: %v", f)
	}
}

func TestFilesAssertions(t *testing.T) {
	c, err := Load([]byte(`
files:
  - expr: files["main.tf"].matches("(?m)^\\s*moved\\s*\\{")
    message: "use a moved block"
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Count() != 1 {
		t.Errorf("a files assertion should count, got %d", c.Count())
	}
	if f := c.RunFiles(map[string][]byte{"main.tf": []byte("resource \"a\" \"b\" {}\n\nmoved {\n  from = a.b\n  to = a.c\n}\n")}); len(f) != 0 {
		t.Errorf("a moved block should pass: %v", f)
	}
	if f := c.RunFiles(map[string][]byte{"main.tf": []byte("# moved { not real }\nresource \"a\" \"c\" {}\n")}); len(f) != 1 {
		t.Errorf("a commented moved block should fail: %v", f)
	}
	if f := c.RunFiles(map[string][]byte{"other.tf": []byte("moved {}")}); len(f) != 1 {
		t.Errorf("a missing file should fail, not crash: %v", f)
	}
}
