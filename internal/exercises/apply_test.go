package exercises

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCleanSourceKeepsTheState(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(filepath.Join(src, "modules", "m"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"main.tf", "old.tf", ".terraform.lock.hcl", "terraform.tfstate", "terraform.tfstate.backup", "terraform.tfstate.d", "modules/m/main.tf"} {
		if err := os.WriteFile(filepath.Join(src, filepath.FromSlash(f)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanSource(src, true); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(src)
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	slices.Sort(left)
	if !slices.Equal(left, []string{"terraform.tfstate", "terraform.tfstate.backup", "terraform.tfstate.d"}) {
		t.Errorf("only the state should survive, got %v", left)
	}
	if err := cleanSource(src, false); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(src); len(entries) != 0 {
		t.Errorf("plan mode keeps nothing, got %v", entries)
	}
	if err := cleanSource(filepath.Join(t.TempDir(), "missing"), true); err != nil {
		t.Errorf("a folder that does not exist yet is fine: %v", err)
	}
}

func TestPendingChanges(t *testing.T) {
	plan := map[string]any{"resource_changes": []any{
		map[string]any{"address": "aws_s3_bucket.a", "change": map[string]any{"actions": []any{"no-op"}}},
		map[string]any{"address": "aws_s3_object.b", "change": map[string]any{"actions": []any{"update"}}},
		map[string]any{"address": "aws_ssm_parameter.c", "change": map[string]any{"actions": []any{"delete", "create"}}},
		map[string]any{"address": "data.aws_caller_identity.d", "change": map[string]any{"actions": []any{"read"}}},
	}}
	got := pendingChanges(plan)
	want := []string{"aws_s3_object.b: update", "aws_ssm_parameter.c: delete, create"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(pendingChanges(map[string]any{})) != 0 {
		t.Error("a plan without resource changes has nothing pending")
	}
}

func TestPreflightOnlyChecksWhatNeedsIt(t *testing.T) {
	bucket := map[string][]byte{"main.tf": []byte(`resource "aws_s3_bucket" "b" {}`)}
	ssm := map[string][]byte{"main.tf": []byte(`resource "aws_ssm_parameter" "p" {}`)}
	if err := preflightHosts(context.Background(), "", bucket); err != nil {
		t.Errorf("no account, no check: %v", err)
	}
	if err := preflightHosts(context.Background(), "123456789012", ssm); err != nil {
		t.Errorf("only buckets need the host name: %v", err)
	}
}

func TestHasSavedState(t *testing.T) {
	home := t.TempDir()
	env := &Env{Home: home}
	if env.HasSavedState() {
		t.Error("nothing is saved yet")
	}
	src := filepath.Join(home, "runs", "buckets1", "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	empty := `{"version":4,"resources":[]}`
	if err := os.WriteFile(filepath.Join(src, "terraform.tfstate"), []byte(empty), 0o644); err != nil {
		t.Fatal(err)
	}
	if env.HasSavedState() {
		t.Error("a state with no resources holds nothing")
	}
	full := `{"version":4,"resources":[{"type":"aws_s3_bucket","name":"b"}]}`
	if err := os.WriteFile(filepath.Join(src, "terraform.tfstate"), []byte(full), 0o644); err != nil {
		t.Fatal(err)
	}
	if !env.HasSavedState() {
		t.Error("a state with a resource is work in progress")
	}
	if err := env.ForgetRun("buckets1"); err != nil {
		t.Fatal(err)
	}
	if env.HasSavedState() {
		t.Error("reset forgets the state")
	}
}

func TestPlanNeedsApply(t *testing.T) {
	outputsOnly := map[string]any{"output_changes": map[string]any{"id": map[string]any{"actions": []any{"create"}}}}
	if !planNeedsApply(outputsOnly) {
		t.Error("a new output has to be written to the state")
	}
	settled := map[string]any{
		"resource_changes": []any{map[string]any{"address": "a.b", "change": map[string]any{"actions": []any{"no-op"}}}},
		"output_changes":   map[string]any{"id": map[string]any{"actions": []any{"no-op"}}},
	}
	if planNeedsApply(settled) {
		t.Error("nothing to do")
	}
	if planNeedsApply(map[string]any{}) {
		t.Error("an empty plan has nothing to apply")
	}
}

func TestHasSavedStateCountsSeededFixExercises(t *testing.T) {
	home := t.TempDir()
	env := &Env{Home: home}
	dir := filepath.Join(home, "runs", "locking1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A fix exercise built without Floci holds nothing in it.
	if err := os.WriteFile(filepath.Join(dir, seededMarker), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if env.HasSavedState() {
		t.Error("a situation built without Floci has nothing in Floci")
	}
	// One built in Floci (a bucket, a lock file) would be lost if Floci were emptied.
	if err := os.WriteFile(filepath.Join(dir, seededMarker), []byte(seededInFloci+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !env.HasSavedState() {
		t.Error("a situation built in Floci is work in progress")
	}
	if err := env.ForgetRun("locking1"); err != nil {
		t.Fatal(err)
	}
	if env.HasSavedState() {
		t.Error("reset forgets it")
	}
}
