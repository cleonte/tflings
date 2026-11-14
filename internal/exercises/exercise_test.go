package exercises

import (
	"strings"
	"testing"
)

func TestParseInfo(t *testing.T) {
	const ok = `
[[exercises]]
name = "a"
path = "exercises/t/a"
topic = "t"
mode = "plan"
hint = "h"
objective = "4e"
engines = ["terraform"]
requires = ["floci"]
timeout = "90s"
`
	all, err := ParseInfo([]byte(ok))
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].RunTimeout().Seconds() != 90 || !all[0].NeedsFloci() {
		t.Fatalf("unexpected parse: %+v", all)
	}

	bad := map[string]string{
		"no name":        "[[exercises]]\npath=\"p\"\ntopic=\"t\"\nmode=\"plan\"\nhint=\"h\"\n",
		"unknown mode":   "[[exercises]]\nname=\"a\"\npath=\"p\"\ntopic=\"t\"\nmode=\"manifest\"\nhint=\"h\"\n",
		"bad objective":  "[[exercises]]\nname=\"a\"\npath=\"p\"\ntopic=\"t\"\nmode=\"plan\"\nhint=\"h\"\nobjective=\"9z\"\n",
		"bad requires":   "[[exercises]]\nname=\"a\"\npath=\"p\"\ntopic=\"t\"\nmode=\"plan\"\nhint=\"h\"\nrequires=[\"kind\"]\n",
		"other engine":   "[[exercises]]\nname=\"a\"\npath=\"p\"\ntopic=\"t\"\nmode=\"plan\"\nhint=\"h\"\nengines=[\"tofu\"]\n",
		"bad timeout":    "[[exercises]]\nname=\"a\"\npath=\"p\"\ntopic=\"t\"\nmode=\"plan\"\nhint=\"h\"\ntimeout=\"soon\"\n",
		"missing hint":   "[[exercises]]\nname=\"a\"\npath=\"p\"\ntopic=\"t\"\nmode=\"plan\"\n",
		"duplicate name": ok + ok,
	}
	for name, doc := range bad {
		if _, err := ParseInfo([]byte(doc)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestEmbeddedOnly(t *testing.T) {
	yes := []string{
		"exercises/t/a/check.yaml",
		"exercises/t/a/setup/state.json",
		"exercises/t/a/setup/nested/seed.tf",
	}
	no := []string{
		"exercises/t/a/main.tf",
		"exercises/t/a/README.md",
		"exercises/t/a/.terraform.lock.hcl",
		"exercises/setup/a/main.tf", // a topic called setup
		"exercises/t/setup/main.tf", // an exercise called setup
	}
	for _, p := range yes {
		if !EmbeddedOnly(p) {
			t.Errorf("%s should stay in the binary", p)
		}
	}
	for _, p := range no {
		if EmbeddedOnly(p) {
			t.Errorf("%s should be written to the workspace", p)
		}
	}
}

func TestSkipReason(t *testing.T) {
	plan := Exercise{Name: "p", Mode: ModePlan}
	apply := Exercise{Name: "a", Mode: ModeApply}
	quiz := Exercise{Name: "q", Mode: ModeQuiz}
	hcp := Exercise{Name: "h", Mode: ModePlan, Requires: []string{RequiresHCP}}

	none := Availability{}
	if quiz.SkipReason(none) != "" {
		t.Error("a quiz needs nothing")
	}
	if !strings.Contains(plan.SkipReason(none), "Terraform") {
		t.Error("a plan exercise needs Terraform")
	}
	if plan.SkipReason(Availability{Terraform: true}) != "" {
		t.Error("a plan exercise without the aws provider does not need Floci")
	}
	if !strings.Contains(apply.SkipReason(Availability{Terraform: true}), "Floci") {
		t.Error("an apply exercise needs Floci")
	}
	if apply.SkipReason(Availability{Terraform: true, Floci: true}) != "" {
		t.Error("apply should run with Terraform and Floci")
	}
	if !strings.Contains(hcp.SkipReason(Availability{Terraform: true}), "HCP") {
		t.Error("an hcp exercise needs a token")
	}
}

func TestMarker(t *testing.T) {
	for _, line := range []string{"# I AM NOT DONE", "  #   I AM NOT DONE", "#I AM NOT DONE"} {
		if !notDoneRegex.MatchString(line) {
			t.Errorf("%q should match", line)
		}
	}
	if notDoneRegex.MatchString("// I AM NOT DONE") {
		t.Error("only # comments carry the marker")
	}
}
