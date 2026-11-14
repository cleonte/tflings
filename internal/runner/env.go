// Package runner runs Terraform for an exercise inside a boundary: a
// from-scratch environment, a private data directory, a wall-clock timeout and
// a cap on output. Nothing the learner has in their own environment reaches
// Terraform, and nothing can reach a real cloud.
package runner

import (
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Floci is where the tflings Floci container listens.
const (
	FlociHost = "127.0.0.1"
	FlociPort = 4577
)

// Endpoint is the generic AWS endpoint: the IP, so that S3 data calls and the
// S3 backend use path style and need no DNS (spikes 03 and 04).
func Endpoint() string { return fmt.Sprintf("http://%s:%d", FlociHost, FlociPort) }

// ControlEndpoint is the endpoint for S3 Control only. The AWS provider
// prepends the account ID to its host, so it must be a name:
// <account>.localhost has to resolve (spike 04, decision 10).
func ControlEndpoint() string { return fmt.Sprintf("http://localhost:%d", FlociPort) }

// Config says what one Terraform run may see.
type Config struct {
	// Account is the 12-digit fake AWS account; empty for runs that need no AWS.
	Account string
	// Home is tflings's own home folder, used as HOME so that nothing is read
	// from the learner's (no ~/.aws, no ~/.terraformrc).
	Home string
	// DataDir is TF_DATA_DIR, one per exercise.
	DataDir string
	// CLIConfig is TF_CLI_CONFIG_FILE (the provider mirror); empty for none.
	CLIConfig string
	// PluginCache is TF_PLUGIN_CACHE_DIR; empty for none.
	PluginCache string
}

// Env builds the environment of a Terraform run from nothing. Only the
// variables below exist; every other AWS_*, TF_*, ARM_*, GOOGLE_*, proxy or
// credential variable of the learner's shell is dropped by not being copied.
func Env(c Config) []string {
	env := []string{
		"PATH=" + safePath(),
		"HOME=" + c.Home,
		"CHECKPOINT_DISABLE=1", // no version-check phone call (also keeps offline runs quiet)
		"TF_IN_AUTOMATION=1",
		"TF_INPUT=0",
		"TF_DATA_DIR=" + c.DataDir,
		"AWS_CONFIG_FILE=" + nullFile(),
		"AWS_SHARED_CREDENTIALS_FILE=" + nullFile(),
	}
	if runtime.GOOS == "windows" {
		// Go and Terraform need these on Windows to find a temp folder and the OS.
		env = append(env, "SystemRoot=C:\\Windows", "TEMP="+filepath.Join(c.Home, "tmp"), "TMP="+filepath.Join(c.Home, "tmp"))
	}
	if c.CLIConfig != "" {
		env = append(env, "TF_CLI_CONFIG_FILE="+c.CLIConfig)
	}
	if c.PluginCache != "" {
		env = append(env, "TF_PLUGIN_CACHE_DIR="+c.PluginCache)
	}
	if c.Account != "" {
		env = append(env,
			"AWS_ACCESS_KEY_ID="+c.Account,
			"AWS_SECRET_ACCESS_KEY=test",
			"AWS_REGION=us-east-1",
			"AWS_ENDPOINT_URL="+Endpoint(),
			"AWS_ENDPOINT_URL_S3_CONTROL="+ControlEndpoint(),
			// Capped retries make a failure quicker but do not bound it (spike 04);
			// the wall-clock timeout in Run does.
			"AWS_MAX_ATTEMPTS=2",
			"AWS_RETRY_MODE=standard",
		)
	}
	return env
}

func nullFile() string {
	if runtime.GOOS == "windows" {
		return "NUL"
	}
	return "/dev/null"
}

// safePath is the PATH of a run: only the system folders, so that a program
// the learner installed (an aws CLI with real credentials, say) cannot be
// found by a local-exec provisioner.
func safePath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32;C:\Windows`
	}
	return "/usr/bin:/bin:/usr/sbin:/sbin"
}

// Lookup returns the value of name in an environment built by Env.
func Lookup(env []string, name string) (string, bool) {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, name+"="); ok {
			return v, true
		}
	}
	return "", false
}

// Names lists the variable names of an environment, sorted.
func Names(env []string) []string {
	var out []string
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}
