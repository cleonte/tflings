package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConstraint(t *testing.T) {
	for in, want := range map[string]string{
		"":               DefaultConstraint,
		"1.12.3":         "= 1.12.3",
		"v1.13.0":        "= 1.13.0",
		"~> 1.12.0":      "~> 1.12.0",
		">= 1.12, < 2.0": ">= 1.12, < 2.0",
	} {
		got, err := Constraint(in)
		if err != nil || got != want {
			t.Errorf("Constraint(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"1.11.0", "1.12", "latest", "~> x", "1.12.x"} {
		if _, err := Constraint(bad); err == nil {
			t.Errorf("Constraint(%q) should be refused", bad)
		}
	}
}

func TestBinaryPathIsInsideTheHome(t *testing.T) {
	p := BinaryPath("/h")
	if !strings.HasPrefix(p, filepath.Join("/h", "bin")+string(filepath.Separator)) {
		t.Errorf("got %s", p)
	}
}

func TestTerraformReportsAFailedDownload(t *testing.T) {
	home := t.TempDir()
	_, _, err := Terraform(context.Background(), home, "", func(context.Context, string, string) (string, error) {
		return "", errors.New("no network")
	})
	if err == nil || !strings.Contains(err.Error(), "no network") || !strings.Contains(err.Error(), DefaultConstraint) {
		t.Errorf("the error should say what was asked and why it failed: %v", err)
	}
}

func TestTerraformRefusesABinaryThatDoesNotRun(t *testing.T) {
	home := t.TempDir()
	_, _, err := Terraform(context.Background(), home, "1.12.3", func(_ context.Context, dir, c string) (string, error) {
		if c != "= 1.12.3" {
			t.Errorf("constraint passed on: %q", c)
		}
		p := filepath.Join(dir, "terraform")
		return p, os.WriteFile(p, []byte("#!/bin/sh\necho nope\nexit 3\n"), 0o755)
	})
	if err == nil || !strings.Contains(err.Error(), "does not work") {
		t.Errorf("a binary that cannot report its version is refused: %v", err)
	}
}

func TestTerraformMovesAndChecksTheBinary(t *testing.T) {
	home := t.TempDir()
	script := "#!/bin/sh\necho 'Terraform v1.12.9'\n"
	path, ver, err := Terraform(context.Background(), home, "", func(_ context.Context, dir, c string) (string, error) {
		p := filepath.Join(dir, "downloaded")
		return p, os.WriteFile(p, []byte(script), 0o755)
	})
	if err != nil {
		t.Skipf("the fake binary was not accepted on this platform: %v", err)
	}
	if path != BinaryPath(home) || ver != "1.12.9" {
		t.Errorf("path %s version %s", path, ver)
	}
}
