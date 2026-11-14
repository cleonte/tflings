package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// A fix exercise's setup would give the cause away: it never reaches the
// learner's folder, not even as empty folders; check.yaml neither.
func TestInitKeepsSetupAndChecksInTheBinary(t *testing.T) {
	src := fstest.MapFS{
		"info.toml":                           {Data: []byte("")},
		"exercises/t/x/README.md":             {Data: []byte("ticket")},
		"exercises/t/x/main.tf":               {Data: []byte("x")},
		"exercises/t/x/check.yaml":            {Data: []byte("x")},
		"exercises/t/x/setup/steps.yaml":      {Data: []byte("x")},
		"exercises/t/x/setup/initial/main.tf": {Data: []byte("x")},
		"exercises/setup/y/main.tf":           {Data: []byte("x")}, // a topic called setup is a topic
	}
	dest := t.TempDir()
	if err := write(src, dest); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"exercises/t/x/README.md", "exercises/t/x/main.tf", "exercises/setup/y/main.tf"} {
		if _, err := os.Stat(filepath.Join(dest, p)); err != nil {
			t.Errorf("%s should be written: %v", p, err)
		}
	}
	for _, p := range []string{"exercises/t/x/check.yaml", "exercises/t/x/setup"} {
		if _, err := os.Stat(filepath.Join(dest, p)); err == nil {
			t.Errorf("%s must stay in the binary", p)
		}
	}
}
