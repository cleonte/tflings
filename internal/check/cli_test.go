package check

import (
	"strings"
	"testing"
)

func load(t *testing.T, y string) *Compiled {
	t.Helper()
	c, err := Load([]byte(y))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const replaceSpec = `
cli:
  subcommand: [apply]
  flags: { replace: "aws_instance.web" }
  allow: { auto-approve: "", var-file: "*" }
  hints: { taint: "taint is deprecated: use -replace on apply" }
`

func TestCheckCommand(t *testing.T) {
	c := load(t, replaceSpec)
	for _, ok := range []string{
		"terraform apply -replace=aws_instance.web",
		"terraform apply -replace aws_instance.web",
		"terraform apply --replace=aws_instance.web -auto-approve",
		"terraform apply -auto-approve -replace=\"aws_instance.web\"",
		"terraform apply -var-file=prod.tfvars -replace=aws_instance.web",
		"terraform apply -var-file prod.tfvars -replace aws_instance.web",
		"  terraform   apply   -replace=aws_instance.web  ",
	} {
		if p := c.CheckCommand(ok); len(p) != 0 {
			t.Errorf("%q should pass, got %v", ok, p)
		}
	}
	for bad, want := range map[string]string{
		"":                                "one line",
		"apply -replace=aws_instance.web": "starts with `terraform`",
		"terraform plan -replace=aws_instance.web":                      "not the right command",
		"terraform taint aws_instance.web":                              "deprecated",
		"terraform apply":                                               "needs the flag -replace",
		"terraform apply -replace=aws_instance.db":                      "not set to the right value",
		"terraform apply -replace=aws_instance.web -lock=false":         "not needed",
		"terraform apply -replace=aws_instance.web extra":               "nothing after the flags",
		"terraform apply -replace=aws_instance.web && ls":               "no pipes",
		"terraform apply -replace=aws_instance.web > out":               "no pipes",
		"terraform apply -auto-approve=false -replace=aws_instance.web": "takes no value",
		"terraform apply \"-replace=aws_instance.web":                   "quote",
	} {
		p := c.CheckCommand(bad)
		if len(p) == 0 || !strings.Contains(strings.Join(p, "\n"), want) {
			t.Errorf("%q: want a message with %q, got %v", bad, want, p)
		}
	}
}

func TestCheckCommandWithAnArgumentAndTwoWordSubcommand(t *testing.T) {
	c := load(t, `
cli:
  subcommand: [state, mv]
  args: [aws_s3_bucket.old, aws_s3_bucket.new]
`)
	if p := c.CheckCommand("terraform state mv aws_s3_bucket.old aws_s3_bucket.new"); len(p) != 0 {
		t.Errorf("should pass: %v", p)
	}
	if p := c.CheckCommand("terraform state mv aws_s3_bucket.new aws_s3_bucket.old"); len(p) == 0 {
		t.Error("the arguments are in order")
	}
	if p := c.CheckCommand("terraform state rm aws_s3_bucket.old"); len(p) == 0 {
		t.Error("another subcommand")
	}
}

func TestCLISpecIsValidatedAtLoad(t *testing.T) {
	for _, bad := range []string{
		"cli:\n  subcommand: []\n",
		"cli:\n  subcommand: [-apply]\n",
		"cli:\n  subcommand: [apply]\n  flags: { \"-out\": x }\n",
		"cli:\n  subcommand: [apply]\n  flags: { out: x }\n  allow: { out: \"*\" }\n",
	} {
		if _, err := Load([]byte(bad)); err == nil {
			t.Errorf("should be refused: %q", bad)
		}
	}
}
