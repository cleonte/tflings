// Package engine is the only place that knows which program runs the
// infrastructure code. Everything else talks to the Engine interface, so that
// OpenTofu can be added later (Phase 8) without touching the runner or the
// exercises.
package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Engine describes one program that runs Terraform-compatible configuration.
type Engine interface {
	// Name is "terraform" (later "tofu"); it is also what lessons print.
	Name() string
	// Binary is the path of the program.
	Binary() string
	// Version is the version the program reports, e.g. "1.12.2".
	Version() string
	// Flags returns the flags that make the command never wait for input and
	// never print colour, as that command accepts them: `validate` and `show`
	// refuse -input, and a flag a command does not know is an error.
	Flags(command string) []string
}

// MinTerraform is the oldest Terraform that tflings supports: the version the
// exam tests (Terraform 1.12).
var MinTerraform = Version{Major: 1, Minor: 12}

// Version is a Terraform version without pre-release parts.
type Version struct{ Major, Minor, Patch int }

func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

// AtLeast reports whether v is the same as or newer than o.
func (v Version) AtLeast(o Version) bool {
	if v.Major != o.Major {
		return v.Major > o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor > o.Minor
	}
	return v.Patch >= o.Patch
}

// ParseVersion reads "1.12.2", "v1.12.2" or "1.13.0-beta1".
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("not a version: %q", s)
	}
	var n [3]int
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return Version{}, fmt.Errorf("not a version: %q", s)
		}
		n[i] = v
	}
	return Version{n[0], n[1], n[2]}, nil
}

type terraform struct {
	binary  string
	version Version
}

func (t *terraform) Name() string    { return "terraform" }
func (t *terraform) Binary() string  { return t.binary }
func (t *terraform) Version() string { return t.version.String() }
func (t *terraform) Flags(command string) []string {
	return terraformFlags(command)
}

// takesInput lists the commands that accept -input=false (verified on
// Terraform 1.12.2 and 1.17.2: validate and show reject it).
var takesInput = map[string]bool{
	"init": true, "plan": true, "apply": true, "destroy": true, "import": true, "refresh": true,
}

// takesNoColor lists the commands that accept -no-color.
var takesNoColor = map[string]bool{
	"init": true, "plan": true, "apply": true, "destroy": true, "import": true, "refresh": true,
	"validate": true, "show": true, "output": true, "fmt": true, "force-unlock": true, "console": true,
}

func terraformFlags(command string) []string {
	var f []string
	if takesNoColor[command] {
		f = append(f, "-no-color")
	}
	if takesInput[command] {
		f = append(f, "-input=false")
	}
	return f
}

// ErrTooOld means the program is a Terraform older than MinTerraform.
var ErrTooOld = errors.New("terraform is older than the version tflings supports")

// NewTerraform checks the program at binary and returns it as an Engine. It
// runs `terraform version -json` with an empty environment, so nothing the
// learner has set can leak into it, and no network is needed.
func NewTerraform(ctx context.Context, binary string) (Engine, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "version", "-json")
	cmd.Env = []string{"CHECKPOINT_DISABLE=1"}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s version: %w", binary, err)
	}
	var info struct {
		Version string `json:"terraform_version"`
	}
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		return nil, fmt.Errorf("%s version -json: %w", binary, err)
	}
	v, err := ParseVersion(info.Version)
	if err != nil {
		return nil, err
	}
	if !v.AtLeast(MinTerraform) {
		return nil, fmt.Errorf("%w: found %s, need %s or newer", ErrTooOld, v, MinTerraform)
	}
	return &terraform{binary: binary, version: v}, nil
}

// Find looks for a usable Terraform: first the tflings-managed copy (managed is
// the path to try, may be empty), then the first terraform on PATH. It returns
// the reason it gave up, for `doctor` to print.
func Find(ctx context.Context, managed string) (Engine, error) {
	var tried []string
	if managed != "" {
		e, err := NewTerraform(ctx, managed)
		if err == nil {
			return e, nil
		}
		tried = append(tried, fmt.Sprintf("%s: %v", managed, err))
	}
	if p, err := exec.LookPath("terraform"); err == nil {
		e, err := NewTerraform(ctx, p)
		if err == nil {
			return e, nil
		}
		tried = append(tried, fmt.Sprintf("%s: %v", p, err))
	} else {
		tried = append(tried, "no terraform on PATH")
	}
	return nil, errors.New(strings.Join(tried, "; "))
}
