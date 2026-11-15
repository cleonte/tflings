package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestPrintErrorSaysWhatWentWrong(t *testing.T) {
	var b bytes.Buffer
	printError(&b, errors.New(`unknown flag: --nope`))
	if !strings.Contains(b.String(), "Error: unknown flag: --nope") || !strings.Contains(b.String(), "--help") {
		t.Errorf("got %q", b.String())
	}
	b.Reset()
	printError(&b, errors.New("no questions for objective \"9z\""))
	if !strings.Contains(b.String(), "no questions") || strings.Contains(b.String(), "--help") {
		t.Errorf("got %q", b.String())
	}
}

func TestPrintErrorSkipsWhatACommandAlreadyReported(t *testing.T) {
	var b bytes.Buffer
	printError(&b, reported(errors.New("No exercise found")))
	if b.Len() != 0 {
		t.Errorf("an error the command printed must not be printed twice, got %q", b.String())
	}
	if reported(nil) != nil {
		t.Error("reported(nil) is nil")
	}
}

func TestUnknownCommandAndFlagFail(t *testing.T) {
	for _, args := range [][]string{{"nope"}, {"--nope"}, {"exam", "--nope"}} {
		c := NewRootCmd("test")
		c.SetArgs(args)
		c.SetOut(&bytes.Buffer{})
		c.SetErr(&bytes.Buffer{})
		if err := c.Execute(); err == nil {
			t.Errorf("%v should fail", args)
		}
	}
}

func TestVersionCommandPrintsTheVersion(t *testing.T) {
	c := NewRootCmd("1.2.3\ncommit: abc")
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetArgs([]string{"version"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "tflings version 1.2.3") || !strings.Contains(out.String(), "commit: abc") {
		t.Errorf("got %q", out.String())
	}
}
