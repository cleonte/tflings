package exercises

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestSplitCommand(t *testing.T) {
	cases := map[string][]string{
		`terraform state mv a b`:                {"terraform", "state", "mv", "a", "b"},
		`terraform state mv 'a.b["x y"]' "c d"`: {"terraform", "state", "mv", `a.b["x y"]`, "c d"},
		`  terraform   apply   -auto-approve  `: {"terraform", "apply", "-auto-approve"},
		`terraform import aws_s3_bucket.a ""`:   {"terraform", "import", "aws_s3_bucket.a", ""},
		`terraform apply $(rm -rf /) | tee x`:   {"terraform", "apply", "$(rm", "-rf", "/)", "|", "tee", "x"},
	}
	for in, want := range cases {
		got, err := splitCommand(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("splitCommand(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := splitCommand(`terraform "oops`); err == nil {
		t.Error("an unterminated quote should be an error")
	}
}

func TestLoadSteps(t *testing.T) {
	ex := Exercise{Name: "x", Path: "exercises/t/x", Mode: ModeFix}
	fsys := func(steps string, extra ...string) fstest.MapFS {
		m := fstest.MapFS{"exercises/t/x/setup/steps.yaml": {Data: []byte(steps)}}
		for _, e := range extra {
			m[e] = &fstest.MapFile{Data: []byte("x")}
		}
		return m
	}
	if _, err := ex.LoadSteps(fsys("steps:\n  - apply: initial\n", "exercises/t/x/setup/initial/main.tf")); err != nil {
		t.Errorf("a valid setup was refused: %v", err)
	}
	bad := map[string]string{
		"no steps":           "steps: []\n",
		"missing folder":     "steps:\n  - apply: gone\n",
		"empty step":         "steps:\n  - {}\n",
		"two things at once": "steps:\n  - apply: initial\n    mutate: {call: ssm.put_parameter, args: {name: a, value: b}}\n",
		"unknown field":      "steps:\n  - apply: initial\n    colour: red\n",
	}
	for name, steps := range bad {
		if _, err := ex.LoadSteps(fsys(steps, "exercises/t/x/setup/initial/main.tf")); err == nil {
			t.Errorf("%s: should be refused", name)
		}
	}
	// A step that changes Floci needs the exercise to say it uses Floci.
	mut := "steps:\n  - mutate: {call: ssm.put_parameter, args: {name: a, value: b}}\n"
	if _, err := ex.LoadSteps(fsys(mut)); err == nil {
		t.Error("a mutation without requires = [floci] should be refused")
	}
	withFloci := ex
	withFloci.Requires = []string{RequiresFloci}
	if _, err := withFloci.LoadSteps(fsys(mut)); err != nil {
		t.Errorf("a mutation with requires = [floci] was refused: %v", err)
	}
	if _, err := withFloci.LoadSteps(fsys("steps:\n  - mutate: {call: iam.delete_everything, args: {}}\n")); err == nil {
		t.Error("a call that is not in the allowlist should be refused")
	}
}
