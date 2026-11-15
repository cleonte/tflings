// Package install downloads Terraform into tflings's own folder, for people who
// do not have it. It uses HashiCorp's hc-install, which checks the SHA256 sums
// of the release and the signature on them. Nothing is installed system-wide,
// and nothing is downloaded unless the user asks for it.
package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cleonte/tflings/internal/engine"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
)

// DefaultConstraint is the Terraform that is installed when none is asked for:
// the newest 1.12.x, the version the exam tests.
const DefaultConstraint = "~> 1.12.0"

// Constraint turns what the user typed into a version constraint: "" is the
// default, "1.12.3" is exactly that version, and anything with an operator
// ("~> 1.12", ">= 1.12, < 2.0") is used as written. It refuses versions older
// than tflings supports when the version is exact.
func Constraint(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultConstraint, nil
	}
	if strings.ContainsAny(s, "<>=~!") || strings.Contains(s, ",") {
		if _, err := version.NewConstraint(s); err != nil {
			return "", fmt.Errorf("not a version constraint: %q", s)
		}
		return s, nil
	}
	v, err := engine.ParseVersion(s)
	if err != nil || strings.Count(s, ".") != 2 {
		return "", fmt.Errorf("not a version: %q (write 1.12.3, or a constraint such as \"~> 1.12.0\")", s)
	}
	if !v.AtLeast(engine.MinTerraform) {
		return "", fmt.Errorf("tflings needs Terraform %s or newer, not %s", engine.MinTerraform, v)
	}
	return "= " + v.String(), nil
}

// BinaryPath is where the managed Terraform lives inside tflings's home.
func BinaryPath(home string) string {
	name := "terraform"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(home, "bin", name)
}

// Installer fetches the newest Terraform that satisfies constraint into dir and
// returns the path of the binary. It is a field so tests do not use the network.
type Installer func(ctx context.Context, dir, constraint string) (string, error)

// Download is the real Installer.
func Download(ctx context.Context, dir, constraint string) (string, error) {
	c, err := version.NewConstraint(constraint)
	if err != nil {
		return "", err
	}
	return (&releases.LatestVersion{
		Product:     product.Terraform,
		Constraints: c,
		InstallDir:  dir,
	}).Install(ctx)
}

// Terraform installs Terraform under home/bin and checks that it runs and is
// new enough. It returns the path and the version reported by the binary.
func Terraform(ctx context.Context, home, constraintText string, get Installer) (path, ver string, err error) {
	constraint, err := Constraint(constraintText)
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	// An older managed copy would be in the way of the new one.
	_ = os.Remove(BinaryPath(home))
	got, err := get(ctx, dir, constraint)
	if err != nil {
		return "", "", fmt.Errorf("could not install Terraform (%s): %w", constraint, err)
	}
	want := BinaryPath(home)
	if got != want {
		if err := os.Rename(got, want); err != nil {
			return "", "", err
		}
	}
	eng, err := engine.NewTerraform(ctx, want)
	if err != nil {
		return "", "", fmt.Errorf("the downloaded Terraform does not work here: %w", err)
	}
	return want, eng.Version(), nil
}
