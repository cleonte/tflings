package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cleonte/tflings/internal/engine"
)

func TestAccountIsStableAndFreshAfterReset(t *testing.T) {
	a := Account("variables1", 0)
	if len(a) != 12 || a[0] == '0' {
		t.Fatalf("bad account %q", a)
	}
	if a != Account("variables1", 0) {
		t.Error("the account must not change between runs")
	}
	if a == Account("variables1", 1) {
		t.Error("a reset must get a fresh account")
	}
	if a == Account("variables2", 0) {
		t.Error("two exercises must not share an account")
	}
}

func TestEnvIsBuiltFromNothing(t *testing.T) {
	// Whatever the learner has in their shell must not get through.
	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAREAL")
	t.Setenv("TF_VAR_secret", "x")
	t.Setenv("HTTPS_PROXY", "http://proxy")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/x")

	env := Env(Config{Account: "123456789012", Home: "/h", DataDir: "/d", PluginCache: "/c", CLIConfig: "/c.tfrc"})
	got := Names(env)
	allowed := []string{
		"AWS_ACCESS_KEY_ID", "AWS_CONFIG_FILE", "AWS_ENDPOINT_URL", "AWS_ENDPOINT_URL_S3_CONTROL",
		"AWS_MAX_ATTEMPTS", "AWS_REGION", "AWS_RETRY_MODE", "AWS_SECRET_ACCESS_KEY",
		"AWS_SHARED_CREDENTIALS_FILE", "CHECKPOINT_DISABLE", "HOME", "PATH", "TF_CLI_CONFIG_FILE",
		"TF_DATA_DIR", "TF_IN_AUTOMATION", "TF_INPUT", "TF_PLUGIN_CACHE_DIR",
	}
	if runtime.GOOS == "windows" {
		// Go and Terraform need these on Windows to find a temp folder and the OS.
		allowed = append(allowed, "SystemRoot", "TEMP", "TMP")
	}
	for _, n := range got {
		if !slices.Contains(allowed, n) {
			t.Errorf("unexpected variable %s", n)
		}
	}
	if v, _ := Lookup(env, "AWS_ACCESS_KEY_ID"); v != "123456789012" {
		t.Errorf("the access key must be the account, got %q", v)
	}
	if v, _ := Lookup(env, "AWS_ENDPOINT_URL"); v != "http://127.0.0.1:4577" {
		t.Errorf("endpoint %q", v)
	}
	if v, _ := Lookup(env, "AWS_ENDPOINT_URL_S3_CONTROL"); v != "http://localhost:4577" {
		t.Errorf("s3 control endpoint %q", v)
	}
	for _, kv := range env {
		if strings.Contains(kv, "AKIAREAL") || strings.Contains(kv, "prod") || strings.Contains(kv, "proxy") {
			t.Errorf("the learner's environment leaked: %s", kv)
		}
	}
}

func TestEnvWithoutAccountHasNoAWS(t *testing.T) {
	env := Env(Config{Home: "/h", DataDir: "/d"})
	for _, n := range Names(env) {
		if strings.HasPrefix(n, "AWS_") && n != "AWS_CONFIG_FILE" && n != "AWS_SHARED_CREDENTIALS_FILE" {
			t.Errorf("a run that needs no AWS must not get %s", n)
		}
	}
}

func TestCLIConfig(t *testing.T) {
	// filepath.ToSlash turns backslashes into slashes only where they separate
	// paths (Windows), so the test uses what the OS considers a separator.
	p := filepath.Join("a", "b c", `q"x`)
	got := CLIConfig(p)
	want := strings.ReplaceAll(filepath.ToSlash(p), `"`, `\"`)
	if !strings.Contains(got, `path    = "`+want+`"`) || !strings.Contains(got, `include = ["hashicorp/aws"]`) || !strings.Contains(got, `exclude = ["hashicorp/aws"]`) {
		t.Errorf("unexpected CLI config:\n%s", got)
	}
}

func TestCheckProviders(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string // substrings of the messages, one per expected problem
	}{
		{"plain region", `provider "aws" { region = "us-east-1" }`, nil},
		{"profile", `provider "aws" {
  region  = "us-east-1"
  profile = "prod"
}`, []string{"profile is not allowed"}},
		{"keys", `provider "aws" {
  access_key = "AKIAIOSFODNN7EXAMPLE"
  secret_key = "x"
}`, []string{"access_key", "secret_key"}},
		{"assume role", `provider "aws" {
  assume_role { role_arn = "arn:aws:iam::123456789012:role/x" }
}`, []string{"assume_role"}},
		{"shared files", `provider "aws" { shared_credentials_files = ["~/.aws/credentials"] }`, []string{"shared_credentials_files"}},
		{"floci endpoints ok", `provider "aws" {
  endpoints {
    s3 = "http://localhost:4577"
    sts = "http://127.0.0.1:4577"
    s3control = "http://000000000001.localhost:4577"
  }
}`, nil},
		{"real endpoint", `provider "aws" {
  endpoints { s3 = "https://s3.amazonaws.com" }
}`, []string{"endpoints.s3 points somewhere other"}},
		{"endpoint from variable", `provider "aws" {
  endpoints { s3 = var.url }
}`, []string{"must be a plain string"}},
		{"wrong port", `provider "aws" { sts_endpoint = "http://localhost:4566" }`, []string{"sts_endpoint points somewhere other"}},
		{"other provider ignored", `provider "random" { profile = "x" }`, nil},
		{"aliased provider is checked", `provider "aws" {
  alias   = "west"
  profile = "p"
}`, []string{"profile"}},
		{"backend", `terraform {
  backend "s3" {
    bucket  = "b"
    profile = "prod"
  }
}`, []string{"profile"}},
		{"backend endpoint ok", `terraform {
  backend "s3" {
    bucket   = "b"
    endpoints = { s3 = "http://127.0.0.1:4577" }
  }
}`, nil},
		{"syntax error is not this check's job", `provider "aws" {`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckProviders(map[string][]byte{"main.tf": []byte(tc.src)})
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %d problems %v", got, len(tc.want), tc.want)
			}
			for i, w := range tc.want {
				if !strings.Contains(got[i].Message, w) {
					t.Errorf("problem %d = %q, want it to contain %q", i, got[i].Message, w)
				}
				if got[i].File != "main.tf" || got[i].Line == 0 {
					t.Errorf("problem %d has no location: %+v", i, got[i])
				}
			}
		})
	}
	if got := CheckProviders(map[string][]byte{"x.tf.json": []byte(`{}`)}); len(got) != 1 {
		t.Error("JSON configuration must be refused")
	}
}

func TestCheckProvidersBackendEndpointsAttribute(t *testing.T) {
	// The s3 backend takes endpoints as an attribute, not a block.
	bad := `terraform {
  backend "s3" {
    bucket    = "b"
    endpoints = { s3 = "https://s3.amazonaws.com", dynamodb = "http://localhost:4577" }
  }
}`
	got := CheckProviders(map[string][]byte{"main.tf": []byte(bad)})
	if len(got) != 1 || !strings.Contains(got[0].Message, "endpoints.s3 points somewhere other") {
		t.Fatalf("got %v", got)
	}
	notObject := "terraform {\n  backend \"s3\" {\n    endpoints = var.e\n  }\n}"
	if got := CheckProviders(map[string][]byte{"main.tf": []byte(notObject)}); len(got) != 1 {
		t.Errorf("a non-literal endpoints value must be refused, got %v", got)
	}
}

// fakeEngine runs /bin/sh (or cmd) instead of Terraform, to test the runner
// without Terraform.
type fakeEngine struct{ bin string }

func (f fakeEngine) Name() string          { return "fake" }
func (f fakeEngine) Binary() string        { return f.bin }
func (f fakeEngine) Version() string       { return "0.0.0" }
func (f fakeEngine) Flags(string) []string { return nil }

func shEngine(t *testing.T) engine.Engine {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}
	return fakeEngine{"/bin/sh"}
}

func TestRunTimeoutKillsTheWholeGroup(t *testing.T) {
	e := shEngine(t)
	marker := filepath.Join(t.TempDir(), "child-survived")
	// The child sleeps in the background and would create the marker afterwards.
	script := "(sleep 2; touch " + marker + ") & sleep 30"
	start := time.Now()
	_, err := Run(context.Background(), Call{Engine: e, Args: []string{"-c", script}, Env: []string{"PATH=/usr/bin:/bin"}, Timeout: 300 * time.Millisecond})
	if _, ok := err.(*TimeoutError); !ok {
		t.Fatalf("expected a TimeoutError, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("timeout took %s", time.Since(start))
	}
	time.Sleep(2500 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Error("a child process outlived the timeout")
	}
}

func TestRunCapsOutputAndKeepsExitCode(t *testing.T) {
	e := shEngine(t)
	out, err := Run(context.Background(), Call{
		Engine: e, Env: []string{"PATH=/usr/bin:/bin"}, MaxOutput: 100,
		Args: []string{"-c", "yes x | head -c 5000; echo oops >&2; exit 2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ExitCode != 2 || !out.Truncated || len(out.Stdout) != 100 || !strings.Contains(string(out.Stderr), "oops") {
		t.Errorf("unexpected output: code=%d truncated=%v stdout=%d stderr=%q", out.ExitCode, out.Truncated, len(out.Stdout), out.Stderr)
	}
}

func TestRunStartsWithAnEmptyEnvironment(t *testing.T) {
	e := shEngine(t)
	t.Setenv("TFLINGS_LEAK_TEST", "leaked")
	out, err := Run(context.Background(), Call{Engine: e, Env: []string{"PATH=/usr/bin:/bin", "ONLY=this"}, Args: []string{"-c", "env | sort"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out.Stdout), "leaked") {
		t.Error("the parent's environment reached the child")
	}
	if !strings.Contains(string(out.Stdout), "ONLY=this") {
		t.Error("the given environment is missing")
	}
}
