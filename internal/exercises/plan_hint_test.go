package exercises

import (
	"errors"
	"strings"
	"testing"

	"github.com/cleonte/tflings/internal/runner"
)

func TestMissingProviderSaysWhatToDo(t *testing.T) {
	stderr := "Error: Failed to query available provider packages\n\nCould not retrieve the list of available versions for provider hashicorp/aws:\nprovider registry.terraform.io/hashicorp/aws was not found in any of the\nsearch locations\n\n  - /home/x/.cache/tflings/plugins\n"
	err := terraformFailed("terraform init could not finish:", runner.Output{Stderr: []byte(stderr)})
	var f *FailedError
	if !errors.As(err, &f) {
		t.Fatalf("want FailedError, got %T", err)
	}
	if !strings.Contains(strings.Join(f.Lines, "\n"), "doctor --prefetch") {
		t.Errorf("the message should say to run doctor --prefetch, got %v", f.Lines)
	}
	var other *FailedError
	if !errors.As(terraformFailed("terraform init could not finish:", runner.Output{Stderr: []byte("Error: Unsupported argument")}), &other) {
		t.Fatal("want FailedError")
	}
	if len(other.Lines) != 1 {
		t.Errorf("other errors get no hint, got %v", other.Lines)
	}
}
