package check

import (
	"fmt"
	"slices"
	"strings"
)

// CLISpec describes the one Terraform command a `cli` exercise accepts. The
// learner's line is split into words by tflings (see SplitCommand) and compared
// with this; it is never run, and never given to a shell.
type CLISpec struct {
	// Subcommand is the words after `terraform`: ["apply"], ["state", "mv"].
	Subcommand []string `json:"subcommand"`
	// Flags are required. The value is what the flag must be set to, or "" for a
	// flag that takes no value. -flag=value and -flag value are both accepted, and
	// one or two dashes.
	Flags map[string]string `json:"flags,omitempty"`
	// Args are the positional arguments after the flags, in order.
	Args []string `json:"args,omitempty"`
	// Allow lists flags that may also be given without being required: "" for a
	// flag without a value, "*" for one with any value.
	Allow map[string]string `json:"allow,omitempty"`
	// Hints say what is wrong when the learner used another subcommand, keyed by
	// the first word of it ("taint": "taint is deprecated ...").
	Hints map[string]string `json:"hints,omitempty"`
}

func (s *CLISpec) validate() error {
	if len(s.Subcommand) == 0 {
		return fmt.Errorf("cli: subcommand is empty")
	}
	for _, w := range s.Subcommand {
		if w == "" || strings.HasPrefix(w, "-") {
			return fmt.Errorf("cli: subcommand words must be words, not %q", w)
		}
	}
	for n := range s.Flags {
		if n == "" || strings.HasPrefix(n, "-") {
			return fmt.Errorf("cli: flag names are written without the dash, not %q", n)
		}
		if _, dup := s.Allow[n]; dup {
			return fmt.Errorf("cli: -%s is both required and allowed", n)
		}
	}
	for n, v := range s.Allow {
		if n == "" || strings.HasPrefix(n, "-") {
			return fmt.Errorf("cli: flag names are written without the dash, not %q", n)
		}
		_ = v
	}
	return nil
}

// SplitCommand splits a line into words, honouring double and single quotes. It
// never runs a shell: there is no expansion, pipe or redirection.
func SplitCommand(line string) ([]string, error) {
	var args []string
	var cur strings.Builder
	var quote rune
	started := false
	for _, ch := range line {
		switch {
		case quote != 0:
			if ch == quote {
				quote = 0
			} else {
				cur.WriteRune(ch)
			}
		case ch == '"' || ch == '\'':
			quote, started = ch, true
		case ch == ' ' || ch == '\t':
			if started || cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("a quote is not closed in %q", line)
	}
	if started || cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args, nil
}

// shellChars are what a shell would act on. They are refused so that a learner
// does not think `a && b` or `a > file` works here: it is one command.
const shellChars = ";|&<>$`()\\\n"

// CheckCommand compares the learner's line with the spec and returns what is
// wrong with it, or nothing when it is right.
func (c *Compiled) CheckCommand(line string) []string {
	s := c.CLI
	if s == nil {
		return []string{"this exercise has no command to check"}
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return []string{"Write the command on one line of answer.sh."}
	}
	if strings.ContainsAny(line, shellChars) {
		return []string{"Write one plain command: no pipes, redirects, `&&`, `;` or `$(...)`. It is not run by a shell."}
	}
	words, err := SplitCommand(line)
	if err != nil {
		return []string{err.Error()}
	}
	if len(words) == 0 || words[0] != "terraform" {
		return []string{"The command starts with `terraform`."}
	}
	words = words[1:]

	// The subcommand is the leading words that are not flags.
	var sub []string
	i := 0
	for i < len(words) && !strings.HasPrefix(words[i], "-") && len(sub) < len(s.Subcommand) {
		sub = append(sub, words[i])
		i++
	}
	if !slices.Equal(sub, s.Subcommand) {
		if len(sub) > 0 {
			if h, ok := s.Hints[sub[0]]; ok {
				return []string{h}
			}
		}
		return []string{fmt.Sprintf("That is not the right command. Think about which terraform subcommand does this (`terraform %s ...` is not it).", strings.Join(sub, " "))}
	}

	// Flags and positional arguments.
	got := map[string]string{}
	var args []string
	var problems []string
	for ; i < len(words); i++ {
		w := words[i]
		if !strings.HasPrefix(w, "-") || w == "-" {
			args = append(args, w)
			continue
		}
		name := strings.TrimLeft(w, "-")
		val, hasVal := "", false
		if k := strings.Index(name, "="); k >= 0 {
			name, val, hasVal = name[:k], name[k+1:], true
		}
		want, required := s.Flags[name]
		anyValue, isAllowed := s.Allow[name]
		if !required && !isAllowed {
			problems = append(problems, fmt.Sprintf("-%s is not needed for this: leave it out.", name))
			continue
		}
		takesValue := (required && want != "") || (!required && anyValue == "*")
		if takesValue && !hasVal {
			if i+1 < len(words) {
				i++
				val = words[i]
				hasVal = true
			}
		}
		if !takesValue && hasVal && val != "true" {
			problems = append(problems, fmt.Sprintf("-%s takes no value.", name))
			continue
		}
		got[name] = val
	}
	names := make([]string, 0, len(s.Flags))
	for n := range s.Flags {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		want := s.Flags[n]
		v, ok := got[n]
		switch {
		case !ok && want == "":
			problems = append(problems, fmt.Sprintf("This needs the flag -%s.", n))
		case !ok:
			problems = append(problems, fmt.Sprintf("This needs the flag -%s with the right value.", n))
		case want != "" && v != want:
			problems = append(problems, fmt.Sprintf("-%s is not set to the right value (you wrote %q).", n, v))
		}
	}
	if !slices.Equal(args, s.Args) {
		if len(s.Args) == 0 {
			problems = append(problems, fmt.Sprintf("There should be nothing after the flags, but there is %q.", strings.Join(args, " ")))
		} else {
			problems = append(problems, fmt.Sprintf("The arguments are not right (expected %d, in the right order).", len(s.Args)))
		}
	}
	return problems
}
