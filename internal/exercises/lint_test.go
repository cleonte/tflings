package exercises_test

import (
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cleonte/tflings"
	"github.com/cleonte/tflings/internal/check"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// TestExerciseLint holds every exercise to the rules of CLAUDE.md, so that a
// contributor finds a mistake in a test and a learner never does.
func TestExerciseLint(t *testing.T) {
	raw, err := fs.ReadFile(tflings.FS, "info.toml")
	if err != nil {
		t.Fatal(err)
	}
	all, err := exercises.ParseInfo(raw)
	if err != nil {
		t.Fatal(err)
	}

	listed := map[string]bool{}
	for _, ex := range all {
		listed[ex.Path] = true
	}
	// Every exercise folder is listed in info.toml, and every listed one exists.
	var onDisk []string
	_ = fs.WalkDir(tflings.FS, "exercises", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && path.Base(p) == exercises.CheckFile {
			onDisk = append(onDisk, path.Dir(p))
		}
		return nil
	})
	sort.Strings(onDisk)
	for _, dir := range onDisk {
		if !listed[dir] {
			t.Errorf("%s has a check.yaml but is not in info.toml", dir)
		}
	}
	for _, ex := range all {
		if !slicesContains(onDisk, ex.Path) {
			t.Errorf("%s is in info.toml but %s has no check.yaml", ex.Name, ex.Path)
		}
	}

	names := map[string]bool{}
	for _, ex := range all {
		ex := ex
		t.Run(ex.Name, func(t *testing.T) {
			if names[ex.Name] {
				t.Error("duplicate name")
			}
			names[ex.Name] = true
			if !strings.HasPrefix(ex.Path, "exercises/"+ex.Topic+"/") {
				t.Errorf("path %s should be under exercises/%s/", ex.Path, ex.Topic)
			}
			if ex.Objective == "" {
				t.Error("every exercise carries the exam objective it practises")
			}
			if ex.Docs == "" {
				t.Error("every exercise links to documentation")
			}
			if _, err := fs.Stat(tflings.FS, path.Join(ex.Path, exercises.ReadmeFile)); err != nil {
				t.Error("no README.md")
			}

			// check.yaml compiles (a CEL mistake is found here, not by a learner).
			cy, err := fs.ReadFile(tflings.FS, ex.CheckPath())
			if err != nil {
				t.Fatal(err)
			}
			c, err := check.Load(cy)
			if err != nil {
				t.Errorf("check.yaml: %v", err)
			} else if ex.Mode == exercises.ModePlan && c.Count() == 0 && !c.Fmt {
				t.Error("a plan exercise needs at least one assertion")
			}

			// The file the learner edits starts with the marker and has a TODO.
			src, err := fs.ReadFile(tflings.FS, ex.Source())
			if err != nil {
				t.Fatalf("no %s", ex.Source())
			}
			if first, _, _ := strings.Cut(string(src), "\n"); !strings.HasPrefix(strings.TrimSpace(first), "# I AM NOT DONE") {
				t.Errorf("%s must start with the line # I AM NOT DONE", ex.Source())
			}
			if !regexp.MustCompile(`#\s*TODO:`).Match(src) {
				t.Errorf("%s needs a # TODO: comment at the fix site", ex.Source())
			}

			// The solution exists and carries neither the marker nor a TODO.
			sol, err := fs.ReadFile(tflings.Solutions, ex.SolutionPath())
			if err != nil {
				t.Fatalf("no solution %s", ex.SolutionPath())
			}
			if regexp.MustCompile(`I AM NOT DONE|#\s*TODO:`).Match(sol) {
				t.Errorf("the solution %s still has the marker or a TODO", ex.SolutionPath())
			}

			// Terraform files of exercise and solution follow the rules.
			files, err := exercises.WorkspaceFiles(tflings.FS, ex.Path)
			if err != nil {
				t.Fatal(err)
			}
			solFiles, err := exercises.WorkspaceFiles(tflings.Solutions, path.Join("solutions", strings.TrimPrefix(ex.Path, "exercises/")))
			if err != nil {
				t.Fatal(err)
			}
			for n, v := range solFiles {
				files["solution:"+n] = v
			}
			usesAWS := false
			for n, data := range files {
				if !strings.HasSuffix(n, ".tf") {
					continue
				}
				if strings.Contains(string(data), `"hashicorp/aws"`) {
					usesAWS = true
				}
				for _, p := range lintTerraform(n, data) {
					t.Error(p)
				}
			}
			if usesAWS && strings.Contains(string(cy), "apply_first") {
				t.Error("apply_first builds the configuration during a plan exercise: it is for configurations that need no cloud")
			}
			if usesAWS {
				if !ex.NeedsFloci() {
					t.Error("an exercise that uses the AWS provider must say requires = [\"floci\"]")
				}
				lock, ok := files[".terraform.lock.hcl"]
				if !ok {
					t.Error("an exercise that uses the AWS provider ships a .terraform.lock.hcl")
				} else if strings.Contains(string(lock), `"h1:`) {
					t.Error("the lock file keeps only zh: hashes (an h1: hash is for one platform)")
				}
			}
		})
	}
}

func slicesContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// lintTerraform applies the rules about resources in an exercise: every bucket
// can be deleted with its contents, and every secret is deleted at once (a
// soft-deleted secret blocks creating the same name again).
func lintTerraform(name string, data []byte) []string {
	f, _ := hclsyntax.ParseConfig(data, name, hcl.InitialPos)
	body, ok := f.Body.(*hclsyntax.Body)
	if !ok {
		return nil
	}
	var out []string
	for _, b := range body.Blocks {
		if b.Type != "resource" || len(b.Labels) != 2 {
			continue
		}
		attrs := b.Body.Attributes
		switch b.Labels[0] {
		case "aws_s3_bucket":
			if !isLiteral(attrs["force_destroy"], "true") {
				out = append(out, name+": aws_s3_bucket."+b.Labels[1]+" needs force_destroy = true")
			}
		case "aws_secretsmanager_secret":
			if !isLiteral(attrs["recovery_window_in_days"], "0") {
				out = append(out, name+": aws_secretsmanager_secret."+b.Labels[1]+" needs recovery_window_in_days = 0")
			}
		}
	}
	return out
}

func isLiteral(a *hclsyntax.Attribute, want string) bool {
	if a == nil {
		return false
	}
	v, diags := a.Expr.Value(nil)
	if diags.HasErrors() {
		return false
	}
	switch want {
	case "true":
		return v.Type().FriendlyName() == "bool" && v.True()
	case "0":
		n, _ := v.AsBigFloat().Int64()
		return v.Type().FriendlyName() == "number" && n == 0
	}
	return false
}

func TestLintTerraformRules(t *testing.T) {
	bad := `
resource "aws_s3_bucket" "a" { bucket = "x" }
resource "aws_s3_bucket" "b" {
  bucket        = "y"
  force_destroy = false
}
resource "aws_secretsmanager_secret" "s" { name = "n" }
`
	if got := lintTerraform("main.tf", []byte(bad)); len(got) != 3 {
		t.Errorf("expected 3 problems, got %v", got)
	}
	good := `
resource "aws_s3_bucket" "a" {
  bucket        = "x"
  force_destroy = true
}
resource "aws_secretsmanager_secret" "s" {
  name                    = "n"
  recovery_window_in_days = 0
}
`
	if got := lintTerraform("main.tf", []byte(good)); len(got) != 0 {
		t.Errorf("unexpected problems: %v", got)
	}
}
