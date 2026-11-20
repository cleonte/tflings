package exercises

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cleonte/tflings/internal/engine"
	"github.com/cleonte/tflings/internal/runner"
)

func TestSandboxEnvIsSealedAndResetChangesTheAccount(t *testing.T) {
	t.Setenv("AWS_PROFILE", "my-real-profile")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAREAL")
	env := &Env{Home: t.TempDir()}
	folder, vars, err := env.SandboxEnv()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(env.Home, "sandbox", "src"); folder != want {
		t.Errorf("folder = %s, want %s", folder, want)
	}
	if st, err := os.Stat(folder); err != nil || !st.IsDir() {
		t.Errorf("the folder should exist: %v", err)
	}
	joined := "\n" + strings.Join(vars, "\n") + "\n"
	first := runner.Account("sandbox", 0)
	for _, want := range []string{"AWS_ACCESS_KEY_ID=" + first, "AWS_ENDPOINT_URL=" + runner.Endpoint(), "AWS_SECRET_ACCESS_KEY=test"} {
		if !strings.Contains(joined, "\n"+want+"\n") {
			t.Errorf("env lacks %s:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "my-real-profile") || strings.Contains(joined, "AKIAREAL") {
		t.Errorf("the learner's AWS settings leaked into the sandbox:\n%s", joined)
	}
	if err := env.SandboxReset(); err != nil {
		t.Fatal(err)
	}
	_, vars, err = env.SandboxEnv()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains("\n"+strings.Join(vars, "\n")+"\n", "AWS_ACCESS_KEY_ID="+first+"\n") {
		t.Error("a reset should give the sandbox a new account")
	}
}

func testEngine(t *testing.T) engine.Engine {
	t.Helper()
	eng, err := engine.Find(context.Background(), os.Getenv("TFLINGS_TEST_TERRAFORM"))
	if err != nil {
		t.Skip("no Terraform found: set TFLINGS_TEST_TERRAFORM")
	}
	return eng
}

func TestSandboxRefusesAnUnsafeConfiguration(t *testing.T) {
	env := &Env{Home: t.TempDir(), Engine: testEngine(t)}
	folder, _, err := env.SandboxEnv()
	if err != nil {
		t.Fatal(err)
	}
	bad := "provider \"aws\" {\n  region  = \"us-east-1\"\n  profile = \"work\"\n}\n"
	if err := os.WriteFile(filepath.Join(folder, "main.tf"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	_, err = env.Sandbox(context.Background(), []string{"plan"}, strings.NewReader(""), &out, &out)
	var failed *FailedError
	if !errors.As(err, &failed) {
		t.Fatalf("a profile should be refused, got %v (output %q)", err, out.String())
	}
	if out.Len() != 0 {
		t.Errorf("Terraform must not have run, it printed %q", out.String())
	}
}

func TestSandboxRefusesAnUnsafeArgument(t *testing.T) {
	env := &Env{Home: t.TempDir(), Engine: testEngine(t)}
	if _, _, err := env.SandboxEnv(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	_, err := env.Sandbox(context.Background(), []string{"init", "-backend-config=endpoint=https://s3.amazonaws.com"}, strings.NewReader(""), &out, &out)
	var failed *FailedError
	if !errors.As(err, &failed) {
		t.Fatalf("a backend endpoint outside Floci should be refused, got %v (output %q)", err, out.String())
	}
	if out.Len() != 0 {
		t.Errorf("Terraform must not have run, it printed %q", out.String())
	}
}

func TestSandboxRunsTerraformAndKeepsItsState(t *testing.T) {
	env := &Env{Home: t.TempDir(), Engine: testEngine(t)}
	folder, _, err := env.SandboxEnv()
	if err != nil {
		t.Fatal(err)
	}
	cfg := "resource \"terraform_data\" \"x\" {\n  input = \"hi\"\n}\n"
	if err := os.WriteFile(filepath.Join(folder, "main.tf"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-input=false"}, {"apply", "-auto-approve", "-input=false"}} {
		var out bytes.Buffer
		code, err := env.Sandbox(context.Background(), args, strings.NewReader(""), &out, &out)
		if err != nil || code != 0 {
			t.Fatalf("terraform %v: code %d, err %v\n%s", args, code, err, out.String())
		}
	}
	if !env.HasSavedState() {
		t.Error("a sandbox with resources in its state is work in progress: Floci must not be emptied under it")
	}
	if err := env.SandboxReset(); err != nil {
		t.Fatal(err)
	}
	if env.HasSavedState() {
		t.Error("after a reset nothing is saved")
	}
	if _, err := os.Stat(filepath.Join(folder, "main.tf")); err == nil {
		t.Error("a reset deletes the sandbox's files")
	}
}
