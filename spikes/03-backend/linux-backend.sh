#!/bin/bash
# S3 backend (existing state from the native runs) from a Linux container: online, then with an unreachable nameserver.
# usage: linux-backend.sh <image> <1.12|latest> <lockfile|ddb> [retry-env: e.g. "AWS_MAX_ATTEMPTS=1 AWS_RETRY_MODE=standard"]
img=$1; v=$2; m=$3; retry=${4:-}; spikes=$(cd "$(dirname "$0")/.." && pwd)
docker run --rm --user root --network host -v "$spikes":/spikes:ro -v tflings-linux-plugins:/cache -e EP="${EP:-http://localhost:4577}" -e V=$v -e M=$m -e RETRY="$retry" "$img" sh -c '
set -u
. /etc/os-release; echo "image: $PRETTY_NAME"
if command -v apt-get >/dev/null; then apt-get update -qq >/dev/null 2>&1; apt-get install -y -qq ca-certificates >/dev/null 2>&1
elif command -v apk >/dev/null; then apk add -q --no-cache ca-certificates >/dev/null 2>&1; fi
mkdir -p /work /home/t && cp /spikes/03-backend/app.tf /spikes/03-backend/backend-*.hcl /work/ && cd /work
tf=/spikes/.cache/bin/terraform-linux-$V; id=$V-$M
run() { env -i PATH=/usr/bin:/bin HOME=/home/t AWS_ACCESS_KEY_ID=000000000031 AWS_SECRET_ACCESS_KEY=test AWS_REGION=us-east-1 \
  AWS_ENDPOINT_URL=$EP AWS_CONFIG_FILE=/dev/null AWS_SHARED_CREDENTIALS_FILE=/dev/null TF_PLUGIN_CACHE_DIR=/cache $RETRY "$tf" "$@" -no-color -input=false 2>&1; }
echo "getent bucket host: $(getent hosts tfstate-spike03.localhost || echo FAIL)"
echo "[online] init:"; run init -backend-config=backend-$M.hcl -backend-config="key=spike03/$id.tfstate" | grep -E "^Error|Successfully configured|lookup" | head -3
echo "[online] plan:"; run plan -detailed-exitcode -var id=$id | grep -E "^Error|No changes|^Plan|lookup" | head -3
printf "nameserver 192.0.2.1\noptions timeout:1 attempts:1\n" > /etc/resolv.conf
echo "[offline] getent bucket host: $(getent hosts tfstate-spike03.localhost || echo FAIL); getent localhost: $(getent hosts localhost | head -1)"
s=$(date +%s); echo "[offline] plan (timeout 120s, retry env: ${RETRY:-default}):"
timeout 120 env -i PATH=/usr/bin:/bin HOME=/home/t AWS_ACCESS_KEY_ID=000000000031 AWS_SECRET_ACCESS_KEY=test AWS_REGION=us-east-1 AWS_ENDPOINT_URL=$EP AWS_CONFIG_FILE=/dev/null AWS_SHARED_CREDENTIALS_FILE=/dev/null TF_PLUGIN_CACHE_DIR=/cache $RETRY "$tf" plan -no-color -input=false -var id=$id 2>&1 | grep -E "^Error|lookup|No changes|^Plan" | head -4 | cut -c1-300
echo "rc=$? elapsed=$(( $(date +%s) - s ))s"
' 2>&1
