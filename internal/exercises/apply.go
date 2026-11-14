package exercises

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cleonte/tflings/internal/awscheck"
	"github.com/cleonte/tflings/internal/check"
	"github.com/cleonte/tflings/internal/runner"
)

// cleanSource empties the scratch folder of an exercise. With keepState (apply
// mode) the Terraform state survives, because it has to survive between runs: an apply exercise is
// worked on in steps, and each save applies on top of what is already there,
// as real work does. Everything else (the .tf files, a stale lock file) is
// rewritten by the run.
func cleanSource(src string, keepState bool) error {
	entries, err := os.ReadDir(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if n := e.Name(); keepState && (n == "terraform.tfstate" || n == "terraform.tfstate.backup") {
			continue
		}
		if err := os.RemoveAll(filepath.Join(src, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// runApply is apply mode. It plans first and applies only if the plan checks
// pass, so a mistake Terraform or the checks can see does not build anything.
// Then it checks the state, and finally that a second plan has nothing to do.
func (e Exercise) runApply(ctx context.Context, env *Env, c *check.Compiled, files map[string][]byte) (Result, error) {
	run, ctx, cancel, err := e.prepare(ctx, env, c, files)
	if err != nil {
		return Result{}, err
	}
	defer cancel()

	if err := preflightHosts(ctx, run.cfg.Account, files); err != nil {
		return Result{}, err
	}

	plan, _, err := run.planFor(ctx, nil, true, false)
	if err != nil {
		return Result{}, err
	}
	if failures := c.RunPlanOnly(plan); len(failures) > 0 {
		return Result{}, &FailedError{Lines: failures}
	}

	// Saving a file that changes nothing should be quick: with nothing to apply,
	// skip the apply and the second plan, which would only repeat this one.
	needsApply := planNeedsApply(plan)
	if needsApply {
		out, err := run.tf(ctx, "apply", nil, []string{filepath.Join(run.dir, "tflings.plan")})
		if err != nil {
			return Result{}, err
		}
		if out.ExitCode != 0 {
			return Result{}, terraformFailed("Terraform could not apply your configuration:", out)
		}
	}

	show, err := run.tf(ctx, "show", []string{"-json"}, nil)
	if err != nil {
		return Result{}, err
	}
	if show.ExitCode != 0 {
		return Result{}, terraformFailed("terraform show could not read the state:", show)
	}
	var state map[string]any
	if err := json.Unmarshal(show.Stdout, &state); err != nil {
		return Result{}, fmt.Errorf("%s: the state is not valid JSON: %w", e.Name, err)
	}
	failures := c.RunConfigOnly(plan)
	failures = append(failures, c.RunState(state)...)
	if len(failures) > 0 {
		return Result{}, &FailedError{Lines: failures}
	}

	if len(c.AWS) > 0 {
		awsFailures, err := c.RunAWS(ctx, awscheck.New(runner.Endpoint(), run.cfg.Account), state)
		if err != nil {
			return Result{}, err
		}
		if len(awsFailures) > 0 {
			return Result{}, &FailedError{Lines: awsFailures}
		}
	}

	if c.EmptyPlanAfter && needsApply {
		again, changed, err := run.planFor(ctx, nil, true, true)
		if err != nil {
			return Result{}, err
		}
		if changed {
			return Result{}, &FailedError{
				Lines:   []string{"Applying worked, but a second plan still wants to change something. A configuration should be stable once it is applied."},
				Details: pendingChanges(again),
			}
		}
	}
	return Result{Exercise: e, Checked: c.Count(), Explanation: c.Explanation}, nil
}

// pendingChanges lists what a plan still wants to do, from its JSON.
func pendingChanges(plan map[string]any) []string {
	var out []string
	rcs, _ := plan["resource_changes"].([]any)
	for _, rc := range rcs {
		m, _ := rc.(map[string]any)
		ch, _ := m["change"].(map[string]any)
		actions, _ := ch["actions"].([]any)
		var as []string
		for _, a := range actions {
			if s, _ := a.(string); s != "no-op" && s != "read" {
				as = append(as, s)
			}
		}
		if len(as) > 0 {
			out = append(out, fmt.Sprintf("%v: %s", m["address"], strings.Join(as, ", ")))
		}
	}
	return out
}

// preflightHosts checks, before an apply that would hang, that the host name
// the AWS provider uses for bucket tags resolves. The provider sends that one
// call to <account>.localhost (spike 04, decision 10); where such names do not
// resolve, the apply would run until the timeout. Only configurations with a
// bucket need it.
func preflightHosts(ctx context.Context, account string, files map[string][]byte) error {
	if account == "" {
		return nil
	}
	uses := false
	for n, data := range files {
		if strings.HasSuffix(n, ".tf") && strings.Contains(string(data), "aws_s3_bucket") {
			uses = true
		}
	}
	if !uses {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	host := account + ".localhost"
	if addrs, err := net.DefaultResolver.LookupHost(ctx, host); err == nil && len(addrs) > 0 {
		return nil
	}
	return &FailedError{
		Lines: []string{"This computer cannot resolve " + host + ", which the AWS provider needs for buckets, so the apply would hang."},
		Details: []string{
			"Add this line to /etc/hosts (it needs sudo), then save again:",
			"    127.0.0.1 " + host,
			"`tflings doctor` explains it and lists the lines for every exercise.",
		},
	}
}

// planNeedsApply says whether applying the plan would do anything: a resource
// to change, or an output whose value changes (a configuration of outputs and
// data sources has no resources, and its state still has to be written).
func planNeedsApply(plan map[string]any) bool {
	if len(pendingChanges(plan)) > 0 {
		return true
	}
	ocs, _ := plan["output_changes"].(map[string]any)
	for _, oc := range ocs {
		m, _ := oc.(map[string]any)
		actions, _ := m["actions"].([]any)
		for _, a := range actions {
			if s, _ := a.(string); s != "no-op" {
				return true
			}
		}
	}
	return false
}
