package exercises

import (
	"strings"
	"testing"

	"github.com/cleonte/tflings/internal/runner"
)

func TestMissingProviderSaysWhatToDo(t *testing.T) {
	stderr := "Error: Failed to query available provider packages\n\nCould not retrieve the list of available versions for provider hashicorp/aws:\nprovider registry.terraform.io/hashicorp/aws was not found in any of the\nsearch locations\n\n  - /home/x/.cache/tflings/plugins\n"
	err := terraformFailed("terraform init could not finish:", runner.Output{Stderr: []byte(stderr)})
	f, ok := err.(*FailedError)
	if !ok {
		t.Fatalf("want FailedError, got %T", err)
	}
	if !strings.Contains(strings.Join(f.Lines, "\n"), "doctor --prefetch") {
		t.Errorf("the message should say to run doctor --prefetch, got %v", f.Lines)
	}
	other := terraformFailed("terraform init could not finish:", runner.Output{Stderr: []byte("Error: Unsupported argument")}).(*FailedError)
	if len(other.Lines) != 1 {
		t.Errorf("other errors get no hint, got %v", other.Lines)
	}
}
