// Package doctor checks that this machine can run the course and says how to
// fix what it cannot. Every failure comes with the command or the line to run.
package doctor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cleonte/tflings/internal/engine"
	"github.com/cleonte/tflings/internal/exercises"
	"github.com/cleonte/tflings/internal/floci"
	"github.com/cleonte/tflings/internal/runner"
)

// Level is how bad a finding is.
type Level int

// The levels of a finding, from fine to blocking.
const (
	OK Level = iota
	Warn
	Fail
)

// Result is one check.
type Result struct {
	Name   string
	Level  Level
	Detail string
	// Fix is what to do about it, empty when Level is OK.
	Fix string
}

// AWSProviderVersion is the AWS provider that `doctor --prefetch` puts in the
// plugin cache and that the exercises' lock files pin (6.73.0 in the spikes).
const AWSProviderVersion = "6.73.0"

// Options is what the checks look at.
type Options struct {
	Env      *exercises.Env
	Floci    *floci.Manager
	InfoFile string
	// Workspace is the folder with info.toml and .tflings.json.
	Workspace string
	// Lookup resolves a host name; net.DefaultResolver by default (a parameter
	// so tests do not need DNS).
	Lookup func(ctx context.Context, host string) ([]string, error)
}

// Run runs every check, in the order a learner should fix them.
func Run(ctx context.Context, o Options) []Result {
	if o.Lookup == nil {
		o.Lookup = net.DefaultResolver.LookupHost
	}
	return []Result{
		checkTerraform(o),
		checkDocker(ctx, o),
		checkFloci(ctx, o),
		checkPluginCache(o),
		checkDNS(ctx, o),
		checkWorkspace(o),
	}
}

// Worst is the most serious level among the results.
func Worst(rs []Result) Level {
	w := OK
	for _, r := range rs {
		if r.Level > w {
			w = r.Level
		}
	}
	return w
}

func checkTerraform(o Options) Result {
	r := Result{Name: "Terraform"}
	if o.Env != nil && o.Env.Engine != nil {
		r.Detail = fmt.Sprintf("%s %s at %s", o.Env.Engine.Name(), o.Env.Engine.Version(), o.Env.Engine.Binary())
		return r
	}
	r.Level = Fail
	r.Detail = "no Terraform " + engine.MinTerraform.String() + " or newer was found"
	r.Fix = "run `tflings doctor --install-terraform` (downloads Terraform into tflings's own folder), or install it from https://developer.hashicorp.com/terraform/install (the exam tests 1.12), or set TFLINGS_TERRAFORM to its path"
	return r
}

func checkDocker(ctx context.Context, o Options) Result {
	r := Result{Name: "Docker"}
	v, err := o.Floci.CheckDocker(ctx)
	switch {
	case err == nil:
		r.Detail = "daemon " + v
	case errors.Is(err, floci.ErrNoDocker):
		r.Level, r.Detail = Warn, "docker is not installed: exercises that build things in Floci are skipped"
		r.Fix = "install Docker Desktop, OrbStack, Colima or Docker Engine; Terraform-only exercises and quizzes still work"
	case errors.Is(err, floci.ErrDockerDown):
		r.Level, r.Detail = Warn, "docker is installed but its daemon is not running: exercises that build things in Floci are skipped"
		r.Fix = "start Docker (Docker Desktop, OrbStack, `colima start`, `limactl start`, or `sudo systemctl start docker`)"
	default:
		r.Level, r.Detail, r.Fix = Warn, err.Error(), "check that `docker version` works"
	}
	return r
}

func checkFloci(ctx context.Context, o Options) Result {
	r := Result{Name: "Floci"}
	if o.Floci.Healthy(ctx) {
		st, err := o.Floci.Current(ctx)
		if err == nil && st.Exists && st.Image != o.Floci.Cfg.Image {
			r.Level = Warn
			r.Detail = fmt.Sprintf("answers on %s, but runs %s, not the pinned image", o.Floci.URL(), st.Image)
			r.Fix = "run `tflings aws up` to replace it (your data is kept)"
			return r
		}
		r.Detail = fmt.Sprintf("answers on %s (Floci %s)", o.Floci.URL(), floci.Version)
		return r
	}
	r.Level = Warn
	r.Detail = "not running on " + o.Floci.URL()
	r.Fix = "run `tflings aws up`"
	return r
}

func checkPluginCache(o Options) Result {
	r := Result{Name: "AWS provider cache"}
	dir := filepath.Join(o.Env.HomeDir(), "plugins", "registry.terraform.io", "hashicorp", "aws")
	entries, err := os.ReadDir(dir)
	var versions []string
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				versions = append(versions, e.Name())
			}
		}
	}
	if len(versions) == 0 {
		r.Level = Warn
		r.Detail = "the AWS provider (about 730 MB) is not downloaded yet"
		r.Fix = "run `tflings doctor --prefetch` once while online; after that, `init` works offline"
		return r
	}
	sort.Strings(versions)
	r.Detail = "hashicorp/aws " + strings.Join(versions, ", ") + " in " + filepath.Dir(dir)
	return r
}

// probeHost is a name under .localhost that nothing special is registered
// for. The AWS provider sends one call (bucket tags) to <account>.localhost, so
// if this resolves, the account names resolve too (spike 04, decision 10).
const probeHost = "tflings-doctor-probe.localhost"

func checkDNS(ctx context.Context, o Options) Result {
	r := Result{Name: "<account>.localhost"}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if addrs, err := o.Lookup(ctx, probeHost); err == nil && len(addrs) > 0 {
		r.Detail = "names under .localhost resolve (" + addrs[0] + ")"
		return r
	}
	r.Level = Warn
	r.Detail = "names under .localhost do not resolve here; exercises that create buckets would hang"
	accounts := FlociAccounts(o)
	if len(accounts) == 0 {
		r.Fix = "when an exercise needs it, add `127.0.0.1 <account>.localhost` lines to /etc/hosts; `tflings doctor` will list the exact lines"
		return r
	}
	r.Fix = HostsFix(accounts)
	return r
}

// HostsFix is the /etc/hosts lines that make the given accounts resolve.
func HostsFix(accounts []string) string {
	var b strings.Builder
	b.WriteString("add these lines to /etc/hosts (needs sudo):\n")
	for _, a := range accounts {
		fmt.Fprintf(&b, "    127.0.0.1 %s.localhost\n", a)
	}
	b.WriteString("  for example: sudo tee -a /etc/hosts")
	return b.String()
}

// FlociAccounts lists the accounts of the exercises in the workspace that talk
// to Floci, as they are right now (a reset changes an exercise's account).
func FlociAccounts(o Options) []string {
	all, err := exercises.List(o.InfoFile)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range all {
		if e.NeedsFloci() {
			out = append(out, exercises.AccountFor(o.Workspace, e.Name, ""))
		}
	}
	return out
}

func checkWorkspace(o Options) Result {
	r := Result{Name: "Exercises folder"}
	all, err := exercises.List(o.InfoFile)
	if err != nil {
		r.Level = Warn
		r.Detail = "no info.toml here, so this is not an exercises folder"
		r.Fix = "run `tflings init` to create one, then cd into it"
		return r
	}
	done, total, _ := exercises.Progress(o.InfoFile)
	needs := 0
	for _, e := range all {
		if e.NeedsFloci() {
			needs++
		}
	}
	r.Detail = fmt.Sprintf("%d exercises, %d done, %d need Floci", total, done, needs)
	return r
}

// Prefetch downloads the AWS provider into the plugin cache, once, by running
// `terraform init` on a scratch configuration that needs it. It needs the
// network. Afterwards `init` of an exercise installs it from the cache with no
// network (see runner.CLIConfig).
func Prefetch(ctx context.Context, env *exercises.Env, progress func(string)) error {
	if env.Engine == nil {
		return errors.New("no Terraform found: run `tflings doctor`")
	}
	home := env.HomeDir()
	dir := filepath.Join(home, "prefetch")
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cfg := fmt.Sprintf("terraform {\n  required_providers {\n    aws = {\n      source  = \"hashicorp/aws\"\n      version = %q\n    }\n  }\n}\n", AWSProviderVersion)
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(cfg), 0o644); err != nil {
		return err
	}
	cache := filepath.Join(home, "plugins")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return err
	}
	rc := runner.Config{Home: filepath.Join(home, "home"), DataDir: filepath.Join(dir, "data"), PluginCache: cache}
	if err := os.MkdirAll(rc.Home, 0o755); err != nil {
		return err
	}
	if progress != nil {
		progress("downloading hashicorp/aws " + AWSProviderVersion + " (about 250 MB, 730 MB unpacked) into " + cache)
	}
	out, err := runner.Run(ctx, runner.Call{
		Engine: env.Engine, Dir: dir, Env: runner.Env(rc),
		Args:    append([]string{"init"}, env.Engine.Flags("init")...),
		Timeout: 15 * time.Minute,
	})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return fmt.Errorf("terraform init failed:\n%s", strings.TrimSpace(string(out.Stderr)))
	}
	return os.RemoveAll(dir)
}
