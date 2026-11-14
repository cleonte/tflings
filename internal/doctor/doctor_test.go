package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cleonte/tflings/internal/exercises"
	"github.com/cleonte/tflings/internal/floci"
	"github.com/cleonte/tflings/internal/runner"
)

// A fake that behaves like a missing docker command.
func missingDockerManager() *floci.Manager {
	m := floci.New()
	m.Docker = floci.CLI{Binary: "/nonexistent/docker"}
	m.Cfg.Port = 1 // nothing answers
	return m
}

const infoWithApply = `
[[exercises]]
name = "buckets1"
path = "exercises/s3/buckets1"
topic = "s3"
mode = "apply"
hint = "h"

[[exercises]]
name = "variables1"
path = "exercises/variables/variables1"
topic = "variables"
mode = "plan"
hint = "h"
`

func workspace(t *testing.T) (dir, info string) {
	t.Helper()
	dir = t.TempDir()
	info = filepath.Join(dir, "info.toml")
	if err := os.WriteFile(info, []byte(infoWithApply), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, info
}

func TestMissingDockerIsAWarningWithAFix(t *testing.T) {
	dir, info := workspace(t)
	env := &exercises.Env{Home: t.TempDir()}
	rs := Run(context.Background(), Options{
		Env: env, Floci: missingDockerManager(), InfoFile: info, Workspace: dir,
		Lookup: func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil },
	})
	byName := map[string]Result{}
	for _, r := range rs {
		byName[r.Name] = r
	}
	if r := byName["Docker"]; r.Level != Warn || !strings.Contains(r.Fix, "install Docker") {
		t.Errorf("docker: %+v", r)
	}
	if r := byName["Terraform"]; r.Level != Fail || !strings.Contains(r.Fix, "developer.hashicorp.com") {
		t.Errorf("terraform: %+v", r)
	}
	if r := byName["Floci"]; r.Level != Warn || !strings.Contains(r.Fix, "tflings aws up") {
		t.Errorf("floci: %+v", r)
	}
	if r := byName["<account>.localhost"]; r.Level != OK {
		t.Errorf("dns: %+v", r)
	}
	if Worst(rs) != Fail {
		t.Error("a missing Terraform is a failure")
	}
	if r := byName["Exercises folder"]; r.Level != OK || !strings.Contains(r.Detail, "2 exercises") || !strings.Contains(r.Detail, "1 need Floci") {
		t.Errorf("workspace: %+v", r)
	}
}

func TestDNSFailurePrintsTheHostsLines(t *testing.T) {
	dir, info := workspace(t)
	rs := Run(context.Background(), Options{
		Env: &exercises.Env{Home: t.TempDir()}, Floci: missingDockerManager(), InfoFile: info, Workspace: dir,
		Lookup: func(context.Context, string) ([]string, error) { return nil, errors.New("no such host") },
	})
	var dns Result
	for _, r := range rs {
		if r.Name == "<account>.localhost" {
			dns = r
		}
	}
	if dns.Level != Warn {
		t.Fatalf("got %+v", dns)
	}
	want := "127.0.0.1 " + runner.Account("buckets1", 0) + ".localhost"
	if !strings.Contains(dns.Fix, want) {
		t.Errorf("the fix must list the account of the exercise that needs Floci:\n%s\nwant a line %q", dns.Fix, want)
	}
	if strings.Contains(dns.Fix, runner.Account("variables1", 0)) {
		t.Error("a plan exercise needs no hosts line")
	}

	// After a reset the exercise has a new account, and the lines change.
	if err := exercises.RecordReset(dir, "buckets1"); err != nil {
		t.Fatal(err)
	}
	if got := FlociAccounts(Options{InfoFile: info, Workspace: dir}); len(got) != 1 || got[0] != runner.Account("buckets1", 1) {
		t.Errorf("after a reset: %v", got)
	}
}

func TestPluginCacheCheck(t *testing.T) {
	home := t.TempDir()
	env := &exercises.Env{Home: home}
	if r := checkPluginCache(Options{Env: env}); r.Level != Warn || !strings.Contains(r.Fix, "--prefetch") {
		t.Errorf("empty cache: %+v", r)
	}
	if err := os.MkdirAll(filepath.Join(home, "plugins", "registry.terraform.io", "hashicorp", "aws", "6.73.0", "linux_arm64"), 0o755); err != nil {
		t.Fatal(err)
	}
	if r := checkPluginCache(Options{Env: env}); r.Level != OK || !strings.Contains(r.Detail, "6.73.0") {
		t.Errorf("filled cache: %+v", r)
	}
}
