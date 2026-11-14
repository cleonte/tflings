package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CLIConfig is the Terraform CLI configuration that makes `init` work with no
// network for the AWS provider (spike 07): the provider is installed from the
// tflings plugin cache, which is already laid out like a filesystem mirror.
// Other providers still come from the registry. The plugin cache alone is not
// enough: Terraform still asks the registry which versions exist. cacheDir must
// not be the TF_PLUGIN_CACHE_DIR of the same run, or Terraform refuses to
// install the provider "to itself".
func CLIConfig(cacheDir string) string {
	p := strings.ReplaceAll(filepath.ToSlash(cacheDir), `"`, `\"`)
	return fmt.Sprintf(`provider_installation {
  filesystem_mirror {
    path    = "%s"
    include = ["hashicorp/aws"]
  }
  direct {
    exclude = ["hashicorp/aws"]
  }
}
`, p)
}

// WriteCLIConfig writes the CLI configuration to path, creating its folder.
func WriteCLIConfig(path, cacheDir string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(CLIConfig(cacheDir)), 0o644)
}
