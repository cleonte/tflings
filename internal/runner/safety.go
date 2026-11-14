package runner

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Problem is one thing in the learner's configuration that tflings refuses to
// run, with where it is.
type Problem struct {
	File    string
	Line    int
	Message string
}

func (p Problem) String() string { return fmt.Sprintf("%s:%d: %s", p.File, p.Line, p.Message) }

// refusedAttrs are arguments of the aws provider and of the s3 backend that
// choose credentials or profiles. tflings sets the (fake) credentials through
// the environment; a configuration that picks its own could reach a real
// account.
var refusedAttrs = map[string]string{
	"profile":                       "a named profile would read the learner's own AWS configuration",
	"shared_config_files":           "reading shared AWS config files is not allowed",
	"shared_credentials_files":      "reading shared AWS credentials files is not allowed",
	"shared_credentials_file":       "reading shared AWS credentials files is not allowed",
	"access_key":                    "tflings sets the credentials for you",
	"secret_key":                    "tflings sets the credentials for you",
	"token":                         "tflings sets the credentials for you",
	"http_proxy":                    "sending traffic through a proxy is not allowed",
	"https_proxy":                   "sending traffic through a proxy is not allowed",
	"custom_ca_bundle":              "a custom CA bundle is not needed for Floci",
	"ec2_metadata_service_endpoint": "the instance metadata service is not available",
}

// refusedBlocks are nested blocks that assume other roles.
var refusedBlocks = map[string]bool{
	"assume_role":                   true,
	"assume_role_with_web_identity": true,
}

// CheckProviders reads the .tf files (name to content) and reports every
// `provider "aws"` block and `backend "s3"` block that could reach anything
// but the tflings Floci. It runs before every plan and apply. It looks only at
// what it can see in the text; the clean environment is the layer underneath,
// and the fake keys being rejected by real AWS is the one below that.
// Files that do not parse are skipped here: Terraform's own validate reports
// the syntax error in its own words.
func CheckProviders(files map[string][]byte) []Problem {
	var problems []Problem
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if strings.HasSuffix(name, ".tf.json") {
			problems = append(problems, Problem{name, 1, "JSON configuration files are not supported in tflings exercises; use .tf"})
			continue
		}
		if filepath.Ext(name) != ".tf" {
			continue
		}
		f, _ := hclsyntax.ParseConfig(files[name], name, hcl.InitialPos)
		body, ok := f.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}
		for _, b := range body.Blocks {
			switch {
			case b.Type == "provider" && len(b.Labels) == 1 && b.Labels[0] == "aws":
				problems = append(problems, checkBody(name, b.Body)...)
			case b.Type == "terraform":
				for _, nb := range b.Body.Blocks {
					if nb.Type == "backend" && len(nb.Labels) == 1 && nb.Labels[0] == "s3" {
						problems = append(problems, checkBody(name, nb.Body)...)
					}
				}
			}
		}
	}
	return problems
}

func checkBody(file string, body *hclsyntax.Body) []Problem {
	var problems []Problem
	attrs := make([]string, 0, len(body.Attributes))
	for n := range body.Attributes {
		attrs = append(attrs, n)
	}
	sort.Strings(attrs)
	for _, n := range attrs {
		a := body.Attributes[n]
		line := a.SrcRange.Start.Line
		if why, bad := refusedAttrs[n]; bad {
			problems = append(problems, Problem{file, line, fmt.Sprintf("%s is not allowed here: %s", n, why)})
			continue
		}
		switch {
		case n == "endpoints":
			// The s3 backend takes endpoints as an object attribute, the provider
			// as a block (below). Every value must be a Floci URL.
			obj, ok := a.Expr.(*hclsyntax.ObjectConsExpr)
			if !ok {
				problems = append(problems, Problem{file, line, "endpoints must be written out as an object of plain strings that point at the tflings Floci (" + Endpoint() + ")"})
				break
			}
			for _, item := range obj.Items {
				key := "endpoints"
				if kv, diags := item.KeyExpr.Value(nil); !diags.HasErrors() && kv.Type().Equals(stringType) && kv.IsKnown() && !kv.IsNull() {
					key = "endpoints." + kv.AsString()
				}
				problems = append(problems, checkEndpoint(file, key, item.ValueExpr, item.ValueExpr.Range().Start.Line)...)
			}
		case isEndpointAttr(n):
			problems = append(problems, checkEndpoint(file, n, a.Expr, line)...)
		}
	}
	for _, nb := range body.Blocks {
		line := nb.TypeRange.Start.Line
		switch {
		case refusedBlocks[nb.Type]:
			problems = append(problems, Problem{file, line, fmt.Sprintf("%s is not allowed here: assuming a role would leave the fake account", nb.Type)})
		case nb.Type == "endpoints":
			names := make([]string, 0, len(nb.Body.Attributes))
			for n := range nb.Body.Attributes {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				a := nb.Body.Attributes[n]
				problems = append(problems, checkEndpoint(file, "endpoints."+n, a.Expr, a.SrcRange.Start.Line)...)
			}
		}
	}
	return problems
}

// isEndpointAttr is true for the older single-endpoint arguments of the s3
// backend and the provider (`endpoint`, `sts_endpoint`, ...).
func isEndpointAttr(n string) bool {
	return n == "endpoint" || strings.HasSuffix(n, "_endpoint")
}

// checkEndpoint accepts an endpoint only if it is a literal URL that points at
// the tflings Floci: http, port 4577, host 127.0.0.1, localhost or a
// subdomain of localhost (the account-ID name of S3 Control).
func checkEndpoint(file, name string, expr hclsyntax.Expression, line int) []Problem {
	lit, diags := expr.Value(nil)
	if diags.HasErrors() || !lit.Type().Equals(stringType) || lit.IsNull() || !lit.IsKnown() {
		return []Problem{{file, line, fmt.Sprintf("%s must be a plain string that points at the tflings Floci (%s)", name, Endpoint())}}
	}
	if !isFlociURL(lit.AsString()) {
		return []Problem{{file, line, fmt.Sprintf("%s points somewhere other than the tflings Floci (%s)", name, Endpoint())}}
	}
	return nil
}

func isFlociURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" {
		return false
	}
	if u.Port() != fmt.Sprint(FlociPort) {
		return false
	}
	h := u.Hostname()
	return h == FlociHost || h == "localhost" || strings.HasSuffix(h, ".localhost")
}
