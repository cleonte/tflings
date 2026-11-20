package exercises

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cleonte/tflings/internal/runner"
)

// The sandbox is a scratch folder for trying Terraform freely, outside the
// exercises, in the same sealed environment: fake credentials, Floci as the only
// endpoint, nothing from the learner's shell. It has one fake AWS account of its
// own (not an exercise's). Its files and state live in <home>/sandbox/src, which
// `reset` empties.
//
//	<home>/sandbox/src     the learner's .tf files and the Terraform state
//	<home>/sandbox/data    TF_DATA_DIR
//	<home>/sandbox/resets  how many times it was reset: part of the account
const sandboxDir = "sandbox"

// sandbox is where the sandbox is and the environment Terraform sees in it.
type sandbox struct {
	dir, src string
	cfg      runner.Config
}

func (e *Env) sandbox() (*sandbox, error) {
	home := e.home()
	dir := filepath.Join(home, sandboxDir)
	s := &sandbox{dir: dir, src: filepath.Join(dir, "src")}
	s.cfg = runner.Config{
		Home:        filepath.Join(home, "home"),
		DataDir:     filepath.Join(dir, "data"),
		Account:     runner.Account("sandbox"+e.AccountSalt, sandboxResets(dir)),
		PluginCache: filepath.Join(dir, "plugin-cache"),
		CLIConfig:   filepath.Join(home, "cli.tfrc"),
	}
	for _, d := range []string{s.cfg.Home, s.src, s.cfg.PluginCache} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	// The AWS provider comes from the prefetched mirror, as for the exercises.
	if err := runner.WriteCLIConfig(s.cfg.CLIConfig, filepath.Join(home, "plugins")); err != nil {
		return nil, err
	}
	return s, nil
}

func sandboxResets(dir string) int {
	data, err := os.ReadFile(filepath.Join(dir, "resets"))
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return n
}

// SandboxEnv is the folder of the sandbox and the environment Terraform sees in
// it, as sorted KEY=value lines, for `tflings sandbox` without arguments.
func (e *Env) SandboxEnv() (folder string, env []string, err error) {
	s, err := e.sandbox()
	if err != nil {
		return "", nil, err
	}
	out := runner.Env(s.cfg)
	sort.Strings(out)
	return s.src, out, nil
}

// Sandbox runs one Terraform command in the sandbox, with the learner's
// terminal attached, and returns Terraform's exit code. The .tf files in the
// folder are checked first, like an exercise's: a configuration that could
// reach anything but the tflings Floci is not run.
func (e *Env) Sandbox(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if e.Engine == nil {
		return -1, &SkippedError{"the sandbox needs Terraform: run `tflings doctor`"}
	}
	s, err := e.sandbox()
	if err != nil {
		return -1, err
	}
	files, err := WorkspaceFiles(os.DirFS(s.src), ".")
	if err != nil {
		return -1, err
	}
	if problems := runner.CheckProviders(files); len(problems) > 0 {
		var ls []string
		for _, p := range problems {
			ls = append(ls, p.String())
		}
		return -1, &FailedError{Lines: append([]string{"tflings will not run this configuration:"}, ls...)}
	}
	if problems := runner.CheckTerraformArgs(args, files); len(problems) > 0 {
		var ls []string
		for _, p := range problems {
			ls = append(ls, p.Message)
		}
		return -1, &FailedError{Lines: append([]string{"tflings will not run this command:"}, ls...)}
	}
	return runner.Attach(ctx, runner.Call{Engine: e.Engine, Dir: s.src, Env: runner.Env(s.cfg), Args: args}, stdin, stdout, stderr)
}

// SandboxReset forgets the sandbox: its files, its state and its account. The
// next use is a fresh folder in a fresh, empty account.
func (e *Env) SandboxReset() error {
	dir := filepath.Join(e.home(), sandboxDir)
	n := sandboxResets(dir) + 1
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "resets"), []byte(fmt.Sprintln(n)), 0o644)
}
