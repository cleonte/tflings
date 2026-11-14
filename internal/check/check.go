// Package check loads and runs the check.yaml of an exercise. check.yaml is
// embedded in the binary and never written to the learner's workspace.
//
// Assertions are CEL expressions (https://cel.dev) over what Terraform itself
// reports: the JSON of the plan (`plan`) and the configuration it contains
// (`config`). A failed or unevaluable assertion shows its message, which is
// written for the learner and must not give the answer away.
package check

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"cel.dev/cel-go/cel"
	"github.com/cleonte/tflings/internal/awscheck"
	"sigs.k8s.io/yaml"
)

// Assertion is one CEL expression that must be true, and what to tell the
// learner when it is not.
type Assertion struct {
	Expr    string `json:"expr"`
	Message string `json:"message"`
}

// AWSAssertion reads something from the fake AWS with one of the typed,
// read-only calls of the awscheck package and asserts on what it returns.
type AWSAssertion struct {
	// Call is the name of the call, e.g. "s3.versioning".
	Call string `json:"call"`
	// Args are the arguments of the call, all strings.
	Args map[string]any `json:"args,omitempty"`
	// Expr sees `result` (what the call returned) and `state`.
	Expr    string `json:"expr"`
	Message string `json:"message"`
}

// File is the content of a check.yaml.
type File struct {
	// Plan assertions see `plan`, the JSON of `terraform show -json <planfile>`
	// (resource_changes, planned_values, output_changes, variables, ...).
	Plan []Assertion `json:"plan,omitempty"`
	// Config assertions see `config`, the "configuration" part of that JSON
	// (root_module.variables, root_module.resources, root_module.outputs, ...).
	Config []Assertion `json:"config,omitempty"`
	// Variants plan the configuration again with other variable values and
	// check that plan too. They catch an answer that hard-codes the value the
	// first plan happens to use.
	Variants []Variant `json:"variants,omitempty"`
	// State assertions (apply mode) see `state`, the JSON of `terraform show -json`
	// for the state after the apply (values.root_module.resources, values.outputs).
	State []Assertion `json:"state,omitempty"`
	// AWS assertions (apply mode) look at what really exists in Floci.
	AWS []AWSAssertion `json:"aws,omitempty"`
	// EmptyPlanAfter (apply mode) requires a second plan to have no changes: the
	// configuration must apply and then be stable.
	EmptyPlanAfter bool `json:"empty_plan_after,omitempty"`
	// Fmt requires `terraform fmt -check` to pass on the learner's files.
	Fmt bool `json:"fmt,omitempty"`

	// Answer is the right choice of a quiz: one letter, or several for a
	// multiple-answer question. Case does not matter.
	Answer []string `json:"answer,omitempty"`
	// Explanation is shown after the exercise passes.
	Explanation string `json:"explanation,omitempty"`
}

// Variant is one extra plan with different input.
type Variant struct {
	// Vars are passed as -var name=value (strings; Terraform converts them to
	// the variable's type).
	Vars   map[string]string `json:"vars"`
	Plan   []Assertion       `json:"plan,omitempty"`
	Config []Assertion       `json:"config,omitempty"`

	plan, config []cel.Program
}

// Compiled is a check.yaml with its expressions compiled.
type Compiled struct {
	File
	plan, config, state []cel.Program
	aws                 []cel.Program
}

// Load parses a check.yaml, compiles every expression and rejects anything
// malformed (unknown fields, a missing message, a non-boolean expression), so
// that a mistake in a check is found by the tests and not by a learner.
func Load(raw []byte) (*Compiled, error) {
	var f File
	if err := yaml.UnmarshalStrict(raw, &f); err != nil {
		return nil, fmt.Errorf("check.yaml: %w", err)
	}
	for i, a := range f.Answer {
		f.Answer[i] = strings.ToLower(strings.TrimSpace(a))
	}
	env, err := cel.NewEnv(
		cel.Variable("plan", cel.DynType),
		cel.Variable("config", cel.DynType),
		cel.Variable("state", cel.DynType),
		cel.Variable("result", cel.DynType),
	)
	if err != nil {
		return nil, err
	}
	c := &Compiled{File: f}
	if c.plan, err = compile(env, f.Plan, "plan"); err != nil {
		return nil, err
	}
	if c.config, err = compile(env, f.Config, "config"); err != nil {
		return nil, err
	}
	if c.state, err = compile(env, f.State, "state"); err != nil {
		return nil, err
	}
	for i, a := range f.AWS {
		if err := awscheck.Validate(a.Call, a.Args); err != nil {
			return nil, fmt.Errorf("check.yaml: aws[%d]: %w", i, err)
		}
		progs, err := compile(env, []Assertion{{Expr: a.Expr, Message: a.Message}}, fmt.Sprintf("aws[%d]", i))
		if err != nil {
			return nil, err
		}
		c.aws = append(c.aws, progs[0])
	}
	for i := range c.Variants {
		v := &c.Variants[i]
		if len(v.Vars) == 0 {
			return nil, fmt.Errorf("check.yaml: variants[%d] needs vars", i)
		}
		if len(v.Plan)+len(v.Config) == 0 {
			return nil, fmt.Errorf("check.yaml: variants[%d] has no assertions", i)
		}
		if v.plan, err = compile(env, v.Plan, fmt.Sprintf("variants[%d].plan", i)); err != nil {
			return nil, err
		}
		if v.config, err = compile(env, v.Config, fmt.Sprintf("variants[%d].config", i)); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func compile(env *cel.Env, as []Assertion, section string) ([]cel.Program, error) {
	var progs []cel.Program
	for i, a := range as {
		if strings.TrimSpace(a.Message) == "" {
			return nil, fmt.Errorf("check.yaml: %s[%d] needs a message", section, i)
		}
		ast, iss := env.Compile(a.Expr)
		if iss.Err() != nil {
			return nil, fmt.Errorf("check.yaml: %s[%d]: %w", section, i, iss.Err())
		}
		if ast.OutputType() != cel.BoolType && ast.OutputType() != cel.DynType {
			return nil, fmt.Errorf("check.yaml: %s[%d] must be a boolean expression", section, i)
		}
		prg, err := env.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("check.yaml: %s[%d]: %w", section, i, err)
		}
		progs = append(progs, prg)
	}
	return progs, nil
}

// Count is how many assertions the check has, variants included.
func (c *Compiled) Count() int {
	n := len(c.plan) + len(c.config) + len(c.state) + len(c.aws)
	if c.EmptyPlanAfter {
		n++
	}
	for _, v := range c.Variants {
		n += len(v.plan) + len(v.config)
	}
	return n
}

// RunPlan evaluates the plan and config assertions against the JSON of a plan
// and returns the messages of those that failed. An expression that cannot be
// evaluated (a field it reads is missing, say) counts as failed.
func (c *Compiled) RunPlan(plan map[string]any) []string {
	return evaluate(plan, group{c.Plan, c.plan}, group{c.Config, c.config})
}

// AWSCaller runs the typed calls of the awscheck package.
type AWSCaller interface {
	Call(ctx context.Context, name string, args map[string]any) (map[string]any, error)
}

// RunAWS makes the calls of the aws assertions and evaluates them, and returns
// the messages of those that failed. Equal calls are made once. An error from a
// call itself (Floci not answering) is returned as an error, not as a failed
// assertion: it is not the learner's mistake.
func (c *Compiled) RunAWS(ctx context.Context, caller AWSCaller, state map[string]any) ([]string, error) {
	var failures []string
	cache := map[string]map[string]any{}
	for i, a := range c.AWS {
		key := a.Call
		if b, err := json.Marshal(a.Args); err == nil {
			key += string(b)
		}
		res, ok := cache[key]
		if !ok {
			var err error
			if res, err = caller.Call(ctx, a.Call, a.Args); err != nil {
				return nil, fmt.Errorf("reading %s from Floci: %w", a.Call, err)
			}
			cache[key] = res
		}
		out, _, err := c.aws[i].Eval(map[string]any{"result": res, "state": state, "plan": map[string]any{}, "config": map[string]any{}})
		if err != nil || out.Value() != true {
			failures = append(failures, a.Message)
		}
	}
	return failures, nil
}

// RunPlanOnly is RunPlan without the config assertions: apply mode checks the
// plan before it applies, and the configuration afterwards, so that an error
// from Terraform or Floci is what the learner sees first.
func (c *Compiled) RunPlanOnly(plan map[string]any) []string {
	return evaluate(plan, group{c.Plan, c.plan})
}

// RunConfigOnly evaluates just the config assertions.
func (c *Compiled) RunConfigOnly(plan map[string]any) []string {
	return evaluate(plan, group{c.Config, c.config})
}

// RunState evaluates the state assertions against the JSON of the state after
// an apply, and returns the messages of those that failed.
func (c *Compiled) RunState(state map[string]any) []string {
	var failures []string
	vars := map[string]any{"state": state, "plan": map[string]any{}, "config": map[string]any{}}
	for i, a := range c.State {
		out, _, err := c.state[i].Eval(vars)
		if err != nil || out.Value() != true {
			failures = append(failures, a.Message)
		}
	}
	return failures
}

// RunVariant is RunPlan for the plan of one variant.
func (c *Compiled) RunVariant(i int, plan map[string]any) []string {
	v := c.Variants[i]
	return evaluate(plan, group{v.Plan, v.plan}, group{v.Config, v.config})
}

type group struct {
	as    []Assertion
	progs []cel.Program
}

func evaluate(plan map[string]any, groups ...group) []string {
	cfg, _ := plan["configuration"].(map[string]any)
	if cfg == nil {
		cfg = map[string]any{}
	}
	vars := map[string]any{"plan": plan, "config": cfg}
	var failures []string
	for _, group := range groups {
		for i, a := range group.as {
			out, _, err := group.progs[i].Eval(vars)
			if err != nil || out.Value() != true {
				failures = append(failures, a.Message)
			}
		}
	}
	return failures
}

// QuizOK reports whether the letters given are exactly the right ones.
func (c *Compiled) QuizOK(given []string) bool {
	if len(c.Answer) == 0 || len(given) != len(c.Answer) {
		return false
	}
	g := make([]string, len(given))
	for i, a := range given {
		g[i] = strings.ToLower(strings.TrimSpace(a))
	}
	slices.Sort(g)
	want := slices.Clone(c.Answer)
	slices.Sort(want)
	return slices.Equal(g, want)
}
