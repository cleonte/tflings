#!/usr/bin/env bash
# Says whether a newer Floci, AWS provider or Terraform exists than the ones
# tflings pins and was proven on. It changes nothing and opens nothing: it
# prints, and exits 1 when something is newer, so a scheduled run shows red.
# Read-only requests to public APIs only; no cloud account is touched.
set -euo pipefail
cd "$(dirname "$0")/.."

pinned_floci=$(sed -n 's/^const Version = "\(.*\)"/\1/p' internal/floci/floci.go)
pinned_aws=$(sed -n 's/^const AWSProviderVersion = "\(.*\)"/\1/p' internal/doctor/doctor.go)
# CLAUDE.md says "proven on Terraform 1.12.2 and 1.17.2": the second is the
# newest one the exercises were proven on.
tested_tf=$(grep -o '1\.12\.2 and 1\.[0-9]*\.[0-9]*' CLAUDE.md | head -1 | awk '{print $3}')

for v in pinned_floci pinned_aws tested_tf; do
  if [ -z "${!v}" ]; then
    echo "::error::could not read $v from the repository"
    exit 2
  fi
done

latest_floci=$(curl -fsS https://api.github.com/repos/floci-io/floci/releases/latest | jq -r .tag_name | sed 's/^v//')
latest_aws=$(curl -fsS https://registry.terraform.io/v1/providers/hashicorp/aws | jq -r .version)
latest_tf=$(curl -fsS https://checkpoint-api.hashicorp.com/v1/check/terraform | jq -r .current_version)

newer=0
report() { # name pinned latest
  if [ "$2" = "$3" ]; then
    echo "ok       $1 $2"
  else
    echo "::warning::$1: pinned or proven on $2, latest is $3"
    newer=1
  fi
}
report "Floci (internal/floci/floci.go)" "$pinned_floci" "$latest_floci"
report "hashicorp/aws (internal/doctor/doctor.go)" "$pinned_aws" "$latest_aws"
report "Terraform (newest version in CLAUDE.md)" "$tested_tf" "$latest_tf"
exit $newer
