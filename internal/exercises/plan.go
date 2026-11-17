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
	unlock, err := env.lockRun(ctx, e.Name)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	run, ctx, cancel, err := e.prepare(ctx, env, c, files)
	if err != nil {
		return Result{}, err
	}
	defer cancel()

	plan, err := run.plan(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	if failures := c.RunPlan(plan); len(failures) > 0 {
		return Result{}, &FailedError{Lines: failures}
	}
	for i, v := range c.Variants {
		if err := run.runVariant(ctx, c, i, v); err != nil {
			return Result{}, err
		}
	}
	return Result{Exercise: e, Checked: c.Count(), Explanation: c.Explanation}, nil
}

// runVariant plans the configuration again with the values of one variant. A
// variant that starts with ApplyFirst builds the configuration with other
// values first, in a state of its own that is deleted afterwards, so that no
// variant sees what another left behind.
func (r *run) runVariant(ctx context.Context, c *check.Compiled, i int, v check.Variant) error {
	if len(v.ApplyFirst) > 0 {
		r.forgetState()
		defer r.forgetState()
		if err := r.applyWith(ctx, v.ApplyFirst); err != nil {
			return err
		}
	}
	about := []string{"(This check " + describeVariant(v) + ".)"}
	if v.ExpectError != "" {
		out, err := r.tf(ctx, "plan", r.varFlags(v.Vars, "-refresh=false"), nil)
		if err != nil {
			return err
		}
		if out.ExitCode == 0 || !strings.Contains(string(out.Stderr)+string(out.Stdout), v.ExpectError) {
			return &FailedError{Lines: []string{v.Message}, Details: about}
		}
		return nil
	}
	plan, _, err := r.planFor(ctx, v.Vars, false, false)
	if err != nil {
		return err
	}
	if failures := c.RunVariant(i, plan); len(failures) > 0 {
		return &FailedError{Lines: failures, Details: about}
	}
	return nil
}

// applyWith builds the configuration with the given values and nothing else:
// the first half of a variant that needs something to exist before it plans.
func (r *run) applyWith(ctx context.Context, vars map[string]string) error {
	if _, _, err := r.planFor(ctx, vars, false, false); err != nil {
		return err
	}
	out, err := r.tf(ctx, "apply", nil, []string{filepath.Join(r.dir, "tflings.plan")})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return terraformFailed("Terraform could not build your configuration for this check:", out)
	}
	return nil
}

// forgetState deletes the Terraform state of the scratch folder.
func (r *run) forgetState() {
	for _, n := range []string{"terraform.tfstate", "terraform.tfstate.backup"} {
		_ = os.Remove(filepath.Join(r.src, n))
	}
}

// varFlags are -var flags for the values, after the given flags.
func (r *run) varFlags(vars map[string]string, first ...string) []string {
	flags := append([]string{}, first...)
	names := make([]string, 0, len(vars))
	for n := range vars {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		flags = append(flags, "-var", n+"="+vars[n])
	}
	return flags
}

func describeVariant(v check.Variant) string {
	if len(v.ApplyFirst) > 0 {
		return "builds your configuration with " + describeVars(v.ApplyFirst) + ", then plans it with " + describeVars(v.Vars)
	}
	return "plans your configuration again with " + describeVars(v.Vars)
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

// plan runs `plan -out` and `show -json` for plan mode: nothing exists yet, so
// there is nothing to refresh.
func (r *run) plan(ctx context.Context, vars map[string]string) (map[string]any, error) {
	plan, _, err := r.planFor(ctx, vars, false, false)
	return plan, err
}

// planFor runs `plan -out` and `show -json` and returns the plan as JSON. With
// detailed it also says whether the plan has changes (exit code 2). With
// refresh Terraform first reads the real state of what the state file knows,
// which apply mode needs and plan mode does not.
func (r *run) planFor(ctx context.Context, vars map[string]string, refresh, detailed bool) (plan map[string]any, changed bool, err error) {
	planFile := filepath.Join(r.dir, "tflings.plan")
	flags := []string{"-out=" + planFile}
	if !refresh {
		flags = append(flags, "-refresh=false")
	}
	if detailed {
		flags = append(flags, "-detailed-exitcode")
	}
	flags = append(flags, r.varFlags(vars)...)
	out, err := r.tf(ctx, "plan", flags, nil)
	if err != nil {
		return nil, false, err
	}
	switch {
	case out.ExitCode == 2 && detailed:
		changed = true
	case out.ExitCode != 0:
		return nil, false, terraformFailed("Terraform could not plan your configuration:", out)
	}
	out, err = r.tf(ctx, "show", []string{"-json"}, []string{planFile})
	if err != nil {
		return nil, false, err
	}
	if out.ExitCode != 0 {
		return nil, false, terraformFailed("terraform show could not read the plan:", out)
	}
	if err := json.Unmarshal(out.Stdout, &plan); err != nil {
		return nil, false, fmt.Errorf("%s: the plan is not valid JSON: %w", r.ex.Name, err)
	}
	return plan, changed, nil
}

// terraformFailed turns a failed Terraform run into something a learner can
// read: Terraform's own error text, trimmed.
func terraformFailed(headline string, out runner.Output) error {
	text := strings.TrimSpace(string(out.Stderr))
	if text == "" {
		text = strings.TrimSpace(string(out.Stdout))
	}
	ls := []string{headline}
	if providerMissing(text) {
		ls = append(ls, "The AWS provider is not in tflings's cache yet. Run `tflings doctor --prefetch` once while online (about 250 MB), then try again.")
	}
	return &FailedError{Lines: ls, Details: lines(stripInitPreamble(text), 40)}
}

// providerMissing says whether init failed because the offline mirror holds no
// AWS provider: the learner has not run `doctor --prefetch`.
func providerMissing(text string) bool {
	return strings.Contains(text, "Failed to query available provider packages") &&
		strings.Contains(text, "was not found in any of the") &&
		strings.Contains(text, "search locations")
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

func lines(s string, limit int) []string {
	ls := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(ls) > limit {
		ls = append(ls[:limit], fmt.Sprintf("... (%d more lines)", len(ls)-limit))
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
	r, err := e.runFor(ex)
	if err != nil {
		return nil, err
	}
	if err := writeSource(r.src, files, ex.Mode == ModeApply || ex.Mode == ModeFix); err != nil {
		return nil, err
	}
	return r, nil
}

// runFor is the run of an exercise without writing any file into its scratch
// folder: where it is and the environment Terraform sees there.
func (e *Env) runFor(ex Exercise) (*run, error) {
	home := e.home()
	dir := filepath.Join(home, "runs", ex.Name)
	src := filepath.Join(dir, "src")
	cfg := runner.Config{Home: filepath.Join(home, "home"), DataDir: filepath.Join(dir, "data")}
	if err := os.MkdirAll(cfg.Home, 0o755); err != nil {
		return nil, err
	}
	if ex.NeedsFloci() {
		cfg.Account = runner.Account(ex.Name+e.AccountSalt, ResetCount(e.workspace(), ex.Name))
		// The AWS provider is installed from the prefetched cache by a filesystem
		// mirror. The mirror folder must not also be TF_PLUGIN_CACHE_DIR: Terraform
		// refuses to install a provider "to itself". The cache is the exercise's
		// own: our lock files hold only zip hashes, which cannot vouch for a cached
		// copy, so Terraform reinstalls it on every init, and for a moment the
		// entry is gone for anyone else reading it (two exercises running at once
		// failed that way). An entry is only a symlink into the mirror, so a
		// cache of one's own costs nothing.
		mirror := filepath.Join(home, "plugins")
		cfg.PluginCache = filepath.Join(dir, "plugin-cache")
		cfg.CLIConfig = filepath.Join(home, "cli.tfrc")
		if err := os.MkdirAll(cfg.PluginCache, 0o755); err != nil {
			return nil, err
		}
		if err := runner.WriteCLIConfig(cfg.CLIConfig, mirror); err != nil {
			return nil, err
		}
	}
	return &run{env: e, ex: ex, dir: dir, src: src, cfg: cfg}, nil
}

// writeSource replaces the .tf files and everything else in a scratch folder
// with files, keeping the Terraform state when keepState says so.
func writeSource(src string, files map[string][]byte, keepState bool) error {
	if err := cleanSource(src, keepState); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p := filepath.Join(src, filepath.FromSlash(n))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, files[n], 0o644); err != nil {
			return err
		}
	}
	if keepState {
		if err := os.MkdirAll(src, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(src, writtenManifest), []byte(strings.Join(names, "\n")+"\n"), 0o644)
	}
	return nil
}

// tf runs one Terraform command: the command, then its non-interactive flags
// as that command accepts them, then extra flags, then positional arguments.
func (r *run) tf(ctx context.Context, command string, flags, positional []string) (runner.Output, error) {
	report(ctx, "terraform "+command)
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

// prepare is what plan and apply mode do first: refuse a configuration that
// could leave the fake AWS, write the files into the scratch folder, check the
// formatting if the exercise asks for it, and run `terraform init`. The
// returned context carries the run's timeout; call cancel when done.
func (e Exercise) prepare(ctx context.Context, env *Env, c *check.Compiled, files map[string][]byte) (*run, context.Context, context.CancelFunc, error) {
	if env.Engine == nil {
		return nil, ctx, nil, &SkippedError{fmt.Sprintf("%s needs Terraform: run `tflings doctor`", e.Name)}
	}
	if len(files) == 0 {
		return nil, ctx, nil, &FailedError{Lines: []string{"There are no Terraform files in " + e.Path + "."}}
	}
	if problems := runner.CheckProviders(files); len(problems) > 0 {
		var lines []string
		for _, p := range problems {
			lines = append(lines, p.String())
		}
		return nil, ctx, nil, &FailedError{Lines: append([]string{"tflings will not run this configuration:"}, lines...)}
	}
	if failures := c.RunFiles(files); len(failures) > 0 {
		return nil, ctx, nil, &FailedError{Lines: failures}
	}
	run, err := env.newRun(e, files)
	if err != nil {
		return nil, ctx, nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, e.RunTimeout())

	if c.Fmt {
		out, err := run.tf(ctx, "fmt", []string{"-check", "-diff", "-recursive"}, nil)
		if err != nil {
			cancel()
			return nil, ctx, nil, err
		}
		if out.ExitCode != 0 {
			cancel()
			return nil, ctx, nil, &FailedError{
				Lines:   []string{"Your files are not formatted the way `terraform fmt` writes them."},
				Details: lines(string(out.Stdout), 40),
			}
		}
	}
	if out, err := run.tf(ctx, "init", nil, nil); err != nil {
		cancel()
		return nil, ctx, nil, err
	} else if out.ExitCode != 0 {
		cancel()
		return nil, ctx, nil, terraformFailed("terraform init could not finish:", out)
	}
	return run, ctx, cancel, nil
}
