package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cleonte/tflings/internal/engine"
)

// terraformForTests finds a Terraform to run: $TFLINGS_TEST_TERRAFORM, else the
// first one on PATH. The test is skipped when there is none (CI installs one).
func terraformForTests(t *testing.T) engine.Engine {
	t.Helper()
	managed := os.Getenv("TFLINGS_TEST_TERRAFORM")
	e, err := engine.Find(context.Background(), managed)
	if err != nil {
		t.Skipf("no usable Terraform: %v", err)
	}
	return e
}

// A configuration that uses only built-in things (terraform_data, locals,
// outputs) needs no provider download and no Floci: init, plan -out and
// show -json must work with a from-scratch environment and no network access.
func TestPlanWithoutAnyProvider(t *testing.T) {
	e := terraformForTests(t)
	dir := t.TempDir()
	home := t.TempDir()
	cfg := `
variable "name" {
  type    = string
  default = "tflings"
}

locals {
  greeting = "hello ${var.name}"
}

resource "terraform_data" "x" {
  input = local.greeting
}

output "greeting" {
  value = terraform_data.x.input
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	// Whatever is in the test runner's own environment must not matter.
	t.Setenv("AWS_PROFILE", "does-not-exist")
	t.Setenv("TF_VAR_name", "from-the-shell")

	env := Env(Config{Home: home, DataDir: filepath.Join(dir, ".data")})
	run := func(args ...string) Output {
		t.Helper()
		out, err := Run(context.Background(), Call{Engine: e, Dir: dir, Env: env, Args: append(args, e.Flags(args[0])...)})
		if err != nil {
			t.Fatal(err)
		}
		if out.ExitCode != 0 {
			t.Fatalf("terraform %v exit %d:\n%s%s", args, out.ExitCode, out.Stdout, out.Stderr)
		}
		return out
	}
	if out := run("init"); !strings.Contains(string(out.Stdout), "built in to Terraform") {
		t.Errorf("expected the built-in provider, got:\n%s", out.Stdout)
	}
	run("validate")
	run("plan", "-out=tfplan")
	show, err := Run(context.Background(), Call{Engine: e, Dir: dir, Env: env, Args: append([]string{"show", "-json"}, append(e.Flags("show"), "tfplan")...)})
	if err != nil || show.ExitCode != 0 {
		t.Fatalf("show -json: %v exit %d %s", err, show.ExitCode, show.Stderr)
	}
	var plan struct {
		ResourceChanges []struct {
			Address string `json:"address"`
			Change  struct {
				Actions []string `json:"actions"`
			} `json:"change"`
		} `json:"resource_changes"`
		Variables map[string]struct {
			Value string `json:"value"`
		} `json:"variables"`
		PlannedValues struct {
			Outputs map[string]struct {
				Value string `json:"value"`
			} `json:"outputs"`
		} `json:"planned_values"`
	}
	if err := json.Unmarshal(show.Stdout, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.ResourceChanges) != 1 || plan.ResourceChanges[0].Address != "terraform_data.x" || plan.ResourceChanges[0].Change.Actions[0] != "create" {
		t.Errorf("unexpected resource changes: %+v", plan.ResourceChanges)
	}
	if got := plan.Variables["name"].Value; got != "tflings" {
		t.Errorf("TF_VAR_name from the shell leaked into the run: variable name = %q", got)
	}
}

// A run that does not finish in time is stopped, with Terraform itself.
func TestTimeoutStopsRealTerraform(t *testing.T) {
	e := terraformForTests(t)
	if os.PathSeparator == '\\' {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	cfg := `
resource "terraform_data" "slow" {
  provisioner "local-exec" {
    command = "sleep 30"
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env(Config{Home: t.TempDir(), DataDir: filepath.Join(dir, ".data")})
	if _, err := Run(context.Background(), Call{Engine: e, Dir: dir, Env: env, Args: append([]string{"init"}, e.Flags("init")...)}); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), Call{
		Engine: e, Dir: dir, Env: env, Timeout: 3_000_000_000,
		Args: append([]string{"apply", "-auto-approve"}, e.Flags("apply")...),
	})
	if _, ok := err.(*TimeoutError); !ok {
		t.Fatalf("expected a TimeoutError, got %v", err)
	}
}
