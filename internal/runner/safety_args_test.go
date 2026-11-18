package runner

import (
	"strings"
	"testing"
)

func TestCheckProvidersReadsBackendConfigFiles(t *testing.T) {
	cases := map[string]struct {
		file, src string
		want      int
	}{
		"clean tfbackend":      {"dev.tfbackend", "bucket = \"b\"\nkey = \"k\"\nregion = \"us-east-1\"\nuse_lockfile = true\n", 0},
		"profile":              {"dev.tfbackend", "profile = \"work\"\n", 1},
		"real endpoint":        {"dev.tfbackend", "endpoint = \"https://s3.amazonaws.com\"\n", 1},
		"floci endpoint":       {"dev.tfbackend", "endpoint = \"http://127.0.0.1:4577\"\n", 0},
		"endpoints object":     {"x.hcl", "endpoints = { s3 = \"https://example.com\" }\n", 1},
		"assume_role argument": {"dev.tfbackend", "assume_role = { role_arn = \"arn:aws:iam::1:role/x\" }\n", 1},
		"lock file untouched":  {".terraform.lock.hcl", "provider \"registry.terraform.io/hashicorp/aws\" {\n  version = \"6.73.0\"\n}\n", 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := CheckProviders(map[string][]byte{tc.file: []byte(tc.src)}); len(got) != tc.want {
				t.Errorf("got %v, want %d problems", got, tc.want)
			}
		})
	}
	// The s3 backend block in a .tf file takes assume_role as an argument too.
	tf := "terraform {\n  backend \"s3\" {\n    bucket = \"b\"\n    assume_role = { role_arn = \"x\" }\n  }\n}\n"
	if got := CheckProviders(map[string][]byte{"main.tf": []byte(tf)}); len(got) != 1 {
		t.Errorf("assume_role as a backend argument should be refused, got %v", got)
	}
}

func TestCheckTerraformArgs(t *testing.T) {
	files := map[string][]byte{
		"main.tf":       []byte(""),
		"ok.tfbackend":  []byte("bucket = \"b\"\n"),
		"bad.tfbackend": []byte("profile = \"x\"\n"),
	}
	refused := [][]string{
		{"init", "-backend-config=endpoint=https://s3.amazonaws.com"},
		{"init", "-backend-config", "endpoints.s3=https://example.com"},
		{"init", "-backend-config=profile=work"},
		{"init", "-backend-config=assume_role={role_arn=\"x\"}"},
		{"init", "-backend-config=bad.tfbackend"},
		{"init", "-backend-config=missing.tfbackend"},
		{"init", "-backend-config=/etc/passwd"},
		{"init", "-backend-config=../x.tfbackend"},
		{"plan", "-var", "s3_endpoint=https://s3.amazonaws.com"},
		{"plan", "-var=api_url=http://evil.example"},
		{"plan", "-var-file=/home/me/.aws/creds.tfvars"},
		{"plan", "-var-file", "../x.tfvars"},
	}
	for _, args := range refused {
		if got := CheckTerraformArgs(args, files); len(got) == 0 {
			t.Errorf("%q should be refused", strings.Join(args, " "))
		}
	}
	allowed := [][]string{
		{"plan"},
		{"apply", "-auto-approve"},
		{"plan", "-var", "tier=large"},
		{"plan", "-var=s3_endpoint=http://127.0.0.1:4577"},
		{"plan", "-var-file=prod.tfvars"},
		{"init", "-backend-config=bucket=my-bucket", "-backend-config=key=state"},
		{"init", "-backend-config=ok.tfbackend"},
		{"init", "-backend-config=endpoint=http://localhost:4577"},
		{"state", "mv", "a.b", "c.d"},
		{"import", "aws_s3_bucket.a", "bucket-name"},
	}
	for _, args := range allowed {
		if got := CheckTerraformArgs(args, files); len(got) != 0 {
			t.Errorf("%q should be allowed, got %v", strings.Join(args, " "), got)
		}
	}
}
