#!/bin/sh
# Run a command with a clean environment pointing only at the local Floci.
# Usage: [ACCOUNT=000000000002] ./env.sh .cache/bin/terraform-1.12 plan
here=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$here/.home" "$here/.cache/plugins"
exec env -i \
  PATH=/usr/bin:/bin:/usr/sbin:/sbin \
  HOME="$here/.home" \
  AWS_ACCESS_KEY_ID="${ACCOUNT:-000000000001}" \
  AWS_SECRET_ACCESS_KEY=test \
  AWS_REGION=us-east-1 \
  AWS_ENDPOINT_URL="${ENDPOINT:-http://localhost:4577}" \
  AWS_CONFIG_FILE=/dev/null \
  AWS_SHARED_CREDENTIALS_FILE=/dev/null \
  TF_PLUGIN_CACHE_DIR="$here/.cache/plugins" \
  "$@"
