package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

const infoNew = `[[exercises]]
name      = "a"
path      = "exercises/t/a"
topic     = "t"
mode      = "plan"
objective = "6b"
hint      = "a better hint"
docs      = "https://example.com/a"

[[exercises]]
name      = "b"
path      = "exercises/t/b"
topic     = "t"
mode      = "plan"
objective = "4c"
hint      = "b"
docs      = "https://example.com/b"
`

// The workspace was made by an older tflings: exercise a had objective 6c.
var infoOld = strings.Replace(strings.Replace(infoNew, `"6b"`, `"6c"`, 1), "a better hint", "a hint", 1)

func setup(t *testing.T) (fstest.MapFS, string) {
	t.Helper()
	src := fstest.MapFS{"info.toml": {Data: []byte(infoNew)}}
	root := t.TempDir()
	for _, n := range []string{"a", "b"} {
		for f, body := range map[string]string{"README.md": "# " + n + "\n", "main.tf": "# I AM NOT DONE\n"} {
			src["exercises/t/"+n+"/"+f] = &fstest.MapFile{Data: []byte(body)}
			p := filepath.Join(root, "exercises", "t", n, f)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(root, "info.toml"), []byte(infoOld), 0o644); err != nil {
		t.Fatal(err)
	}
	return src, root
}

func TestUpdateRefreshesTheInfoOfExercisesYouHave(t *testing.T) {
	src, root := setup(t)

	res, err := Update(src, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Info) != 1 || res.Info[0] != "a" {
		t.Fatalf("only a changed: %v", res.Info)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "info.toml")); string(got) != infoOld {
		t.Error("a dry run must not write anything")
	}

	res, err = Update(src, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Info) != 1 || len(res.Added) != 0 {
		t.Fatalf("result: %+v", res)
	}
	got, _ := os.ReadFile(filepath.Join(root, "info.toml"))
	if !strings.Contains(string(got), `objective = "6b"`) || !strings.Contains(string(got), "a better hint") {
		t.Errorf("a should have the binary's entry:\n%s", got)
	}
	bak, _ := os.ReadFile(filepath.Join(root, "info.toml.bak"))
	if string(bak) != infoOld {
		t.Error("the earlier info.toml is kept as info.toml.bak")
	}

	// A second update has nothing left to do.
	res, err = Update(src, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Info) != 0 || len(res.Added) != 0 {
		t.Errorf("up to date now: %+v", res)
	}
}

func TestUpdateKeepsEntriesThatOnlyTheWorkspaceHas(t *testing.T) {
	src, root := setup(t)
	extra := "\n[[exercises]]\nname      = \"mine\"\npath      = \"exercises/t/mine\"\ntopic     = \"t\"\nmode      = \"plan\"\nobjective = \"1a\"\nhint      = \"h\"\ndocs      = \"d\"\n"
	if err := os.WriteFile(filepath.Join(root, "info.toml"), []byte(infoOld+extra), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(src, root, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "info.toml"))
	if !strings.Contains(string(got), `name      = "mine"`) {
		t.Errorf("an exercise the binary does not know stays:\n%s", got)
	}
}
