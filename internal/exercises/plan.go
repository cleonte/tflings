package exercises

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cleonte/tflings/internal/check"
	"github.com/cleonte/tflings/internal/runner"
)

// WorkspaceFiles reads the files of an exercise folder that Terraform needs
// (everything except the lesson and what stays in the binary), keyed by path
// relative to the folder with forward slashes. Hidden folders such as .terraform
// are skipped; .terraform.lock.hcl is a file and is kept.
func WorkspaceFiles(fsys fs.FS, dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, dir), "/")
		if d.IsDir() {
			if rel != "" && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() == ReadmeFile || EmbeddedOnly(p) || strings.HasSuffix(d.Name(), "~") {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		files[rel] = data
		return nil
	})
	return files, err
}

// ProofFiles is what a plan exercise looks like with its solution in place:
// the exercise's own files, with those the solution folder has replaced.
func (e Exercise) ProofFiles(env *Env) (map[string][]byte, error) {
	files, err := WorkspaceFiles(env.Embedded, e.Path)
	if err != nil {
		return nil, err
	}
	sol, err := WorkspaceFiles(env.Solutions, path.Join("solutions", strings.TrimPrefix(e.Path, "exercises/")))
	if err != nil {
		return nil, err
	}
	for k, v := range sol {
		files[k] = v
	}
	return files, nil
}

// runPlan is plan mode: Terraform plans the learner's configuration and the
// assertions of check.yaml look at the plan and at the configuration inside it.
// Nothing is created anywhere.
func (e Exercise) runPlan(ctx context.Context, env *Env, c *check.Compiled, files map[string][]byte) (Result, error) {
	if env.Engine == nil {
		return Result{}, &SkippedError{fmt.Sprintf("%s needs Terraform: run `tflings doctor`", e.Name)}
	}
	if len(files) == 0 {
		return Result{}, &FailedError{Lines: []string{"There are no Terraform files in " + e.Path + "."}}
	}

	if problems := runner.CheckProviders(files); len(problems) > 0 {
		var lines []string
		for _, p := range problems {
			lines = append(lines, p.String())
		}
		return Result{}, &FailedError{Lines: append([]string{"tflings will not run this configuration:"}, lines...)}
	}

	run, err := env.newRun(e, files)
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, e.RunTimeout())
	defer cancel()

	if c.Fmt {
		out, err := run.tf(ctx, "fmt", []string{"-check", "-diff", "-recursive"}, nil)
		if err != nil {
			return Result{}, err
		}
		if out.ExitCode != 0 {
			return Result{}, &FailedError{
				Lines:   []string{"Your files are not formatted the way `terraform fmt` writes them."},
				Details: lines(string(out.Stdout), 40),
			}
		}
	}

	if out, err := run.tf(ctx, "init", nil, nil); err != nil {
		return Result{}, err
	} else if out.ExitCode != 0 {
		return Result{}, terraformFailed("terraform init could not finish:", out)
	}

	plan, err := run.plan(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	if failures := c.RunPlan(plan); len(failures) > 0 {
		return Result{}, &FailedError{Lines: failures}
	}
	for i, v := range c.Variants {
		plan, err := run.plan(ctx, v.Vars)
		if err != nil {
			return Result{}, err
		}
		if failures := c.RunVariant(i, plan); len(failures) > 0 {
			return Result{}, &FailedError{
				Lines:   failures,
				Details: []string{"(This check plans your configuration again with " + describeVars(v.Vars) + ".)"},
			}
		}
	}
	return Result{Exercise: e, Checked: c.Count(), Explanation: c.Explanation}, nil
}

func describeVars(vars map[string]string) string {
	names := make([]string, 0, len(vars))
	for n := range vars {
		names = append(names, n)
	}
	sort.Strings(names)
	var parts []string
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("%s = %q", n, vars[n]))
	}
	return strings.Join(parts, ", ")
}

// plan runs `plan -out` and `show -json` and returns the plan as JSON.
func (r *run) plan(ctx context.Context, vars map[string]string) (map[string]any, error) {
	planFile := filepath.Join(r.dir, "tflings.plan")
	flags := []string{"-out=" + planFile, "-refresh=false"}
	names := make([]string, 0, len(vars))
	for n := range vars {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		flags = append(flags, "-var", n+"="+vars[n])
	}
	if out, err := r.tf(ctx, "plan", flags, nil); err != nil {
		return nil, err
	} else if out.ExitCode != 0 {
		return nil, terraformFailed("Terraform could not plan your configuration:", out)
	}
	out, err := r.tf(ctx, "show", []string{"-json"}, []string{planFile})
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, terraformFailed("terraform show could not read the plan:", out)
	}
	var plan map[string]any
	if err := json.Unmarshal(out.Stdout, &plan); err != nil {
		return nil, fmt.Errorf("%s: the plan is not valid JSON: %w", r.ex.Name, err)
	}
	return plan, nil
}

// terraformFailed turns a failed Terraform run into something a learner can
// read: Terraform's own error text, trimmed.
func terraformFailed(headline string, out runner.Output) error {
	text := strings.TrimSpace(string(out.Stderr))
	if text == "" {
		text = strings.TrimSpace(string(out.Stdout))
	}
	return &FailedError{Lines: []string{headline}, Details: lines(stripInitPreamble(text), 40)}
}

// initPreamble is the first error block of a failed `terraform init` whose
// cause is the configuration itself: it only says that the next block is the
// real one.
const initPreamble = "Error: Terraform encountered problems during initialisation"

// stripInitPreamble drops that generic first block, so the learner sees the
// real error first.
func stripInitPreamble(text string) string {
	if !strings.HasPrefix(text, initPreamble) {
		return text
	}
	if i := strings.Index(text[len(initPreamble):], "\nError: "); i >= 0 {
		return strings.TrimSpace(text[len(initPreamble)+i:])
	}
	return text
}

func lines(s string, max int) []string {
	ls := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(ls) > max {
		ls = append(ls[:max], fmt.Sprintf("... (%d more lines)", len(ls)-max))
	}
	return ls
}

// A run is one exercise's scratch folder and the environment Terraform sees.
type run struct {
	env *Env
	ex  Exercise
	dir string // <home>/runs/<name>: src/ holds the files, data/ is TF_DATA_DIR
	src string
	cfg runner.Config
	// deadline is shared by every step of the run.
}

// newRun writes the exercise's files into a fresh scratch folder. The learner's
// own folder is never used as a working directory, so Terraform's lock file and
// data directory do not appear in it.
func (e *Env) newRun(ex Exercise, files map[string][]byte) (*run, error) {
	home := e.home()
	dir := filepath.Join(home, "runs", ex.Name)
	src := filepath.Join(dir, "src")
	if err := os.RemoveAll(src); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p := filepath.Join(src, filepath.FromSlash(n))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, files[n], 0o644); err != nil {
			return nil, err
		}
	}
	cfg := runner.Config{Home: filepath.Join(home, "home"), DataDir: filepath.Join(dir, "data")}
	if err := os.MkdirAll(cfg.Home, 0o755); err != nil {
		return nil, err
	}
	if ex.NeedsFloci() {
		cfg.Account = runner.Account(ex.Name, ResetCount(e.workspace(), ex.Name))
		cfg.PluginCache = filepath.Join(home, "plugins")
		cfg.CLIConfig = filepath.Join(home, "cli.tfrc")
		if err := os.MkdirAll(cfg.PluginCache, 0o755); err != nil {
			return nil, err
		}
		if err := runner.WriteCLIConfig(cfg.CLIConfig, cfg.PluginCache); err != nil {
			return nil, err
		}
	}
	return &run{env: e, ex: ex, dir: dir, src: src, cfg: cfg}, nil
}

// tf runs one Terraform command: the command, then its non-interactive flags
// as that command accepts them, then extra flags, then positional arguments.
func (r *run) tf(ctx context.Context, command string, flags, positional []string) (runner.Output, error) {
	args := append([]string{command}, r.env.Engine.Flags(command)...)
	args = append(args, flags...)
	args = append(args, positional...)
	timeout := time.Minute
	if dl, ok := ctx.Deadline(); ok {
		timeout = time.Until(dl)
	}
	out, err := runner.Run(ctx, runner.Call{
		Engine: r.env.Engine, Dir: r.src, Env: runner.Env(r.cfg), Args: args, Timeout: timeout,
	})
	var te *runner.TimeoutError
	if errors.As(err, &te) {
		return out, fmt.Errorf("%s: %w. If this machine has no network, run `tflings doctor`", r.ex.Name, err)
	}
	return out, err
}
