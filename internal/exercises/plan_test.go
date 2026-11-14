package exercises

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestStripInitPreamble(t *testing.T) {
	in := "Error: Terraform encountered problems during initialisation, including problems\nwith the configuration, described below.\n\nThe Terraform configuration must be valid.\n\n\nError: Invalid multi-line string\n\n  on main.tf line 9"
	got := stripInitPreamble(in)
	if !strings.HasPrefix(got, "Error: Invalid multi-line string") || strings.Contains(got, "must be valid") {
		t.Errorf("preamble not removed:\n%s", got)
	}
	other := "Error: Failed to query available provider packages\n\ncould not connect"
	if stripInitPreamble(other) != other {
		t.Error("other errors must be left alone")
	}
}

func TestWorkspaceFilesSkipsWhatDoesNotBelong(t *testing.T) {
	fsys := fstest.MapFS{
		"exercises/t/a/main.tf":             {Data: []byte("a")},
		"exercises/t/a/modules/m/main.tf":   {Data: []byte("m")},
		"exercises/t/a/.terraform.lock.hcl": {Data: []byte("lock")},
		"exercises/t/a/.terraform/x":        {Data: []byte("data dir")},
		"exercises/t/a/README.md":           {Data: []byte("lesson")},
		"exercises/t/a/check.yaml":          {Data: []byte("answers")},
		"exercises/t/a/setup/seed.json":     {Data: []byte("seed")},
		"exercises/t/b/main.tf":             {Data: []byte("other exercise")},
	}
	got, err := WorkspaceFiles(fsys, "exercises/t/a")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"main.tf", "modules/m/main.tf", ".terraform.lock.hcl"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, w := range want {
		if _, ok := got[w]; !ok {
			t.Errorf("missing %s in %v", w, got)
		}
	}
}
