#!/bin/bash
# Endpoint env var REMOVED, fake keys only. Expect an auth failure from real AWS and no state.
# usage: safety.sh <1.12|latest>
v=$1; here=$(cd "$(dirname "$0")" && pwd); h=$here/..
w=$here/work-safety-$v; rm -rf "$w"; mkdir -p "$w" && cp "$here/main.tf" "$w/" && cd "$w"
run() { env -i PATH=/usr/bin:/bin HOME="$h/.home" AWS_ACCESS_KEY_ID=000000000001 AWS_SECRET_ACCESS_KEY=test \
  AWS_REGION=us-east-1 AWS_CONFIG_FILE=/dev/null AWS_SHARED_CREDENTIALS_FILE=/dev/null \
  TF_PLUGIN_CACHE_DIR="$h/.cache/plugins" "$h/.cache/bin/terraform-$v" "$@" -no-color -input=false 2>&1; }
run init >/dev/null
for c in "plan" "apply -auto-approve"; do echo "== $c"; run $c | grep -vE '^\s*$' | head -8; done
echo "== files:"; ls -A
