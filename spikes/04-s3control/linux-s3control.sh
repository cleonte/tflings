#!/bin/bash
# usage: [GEN=url] [S3C=url|""] [HOSTS=0|1] [OFFLINE=0|1] [RETRY="AWS_MAX_ATTEMPTS=1 AWS_RETRY_MODE=standard"] linux-s3control.sh <image> <1.12|latest>
# Spike 01 config (+force_destroy) in a Linux container (host network = Lima VM loopback, Floci on :4577), account 000000000041.
img=$1; v=$2; spikes=$(cd "$(dirname "$0")/.." && pwd)
docker run --rm --user root --network host -v "$spikes":/spikes:ro -v tflings-linux-plugins:/cache \
  -e V=$v -e GEN="${GEN:-http://localhost:4577}" -e S3C="${S3C:-}" -e HOSTS="${HOSTS:-0}" -e OFFLINE="${OFFLINE:-0}" -e RETRY="${RETRY:-}" -e APPLY_TO="${APPLY_TO:-400}" -e DESTROY_TO="${DESTROY_TO:-300}" "$img" sh -c '
set -u
if command -v apt-get >/dev/null; then apt-get update -qq >/dev/null 2>&1; apt-get install -y -qq ca-certificates >/dev/null 2>&1
elif command -v apk >/dev/null; then apk add -q --no-cache ca-certificates >/dev/null 2>&1; fi
mkdir -p /work /home/t && cp /spikes/04-s3control/main.tf /work/ && cd /work
tf=/spikes/.cache/bin/terraform-linux-$V
s3c=""; [ -n "$S3C" ] && s3c="AWS_ENDPOINT_URL_S3_CONTROL=$S3C"
run() { env -i PATH=/usr/bin:/bin HOME=/home/t AWS_ACCESS_KEY_ID=000000000041 AWS_SECRET_ACCESS_KEY=test AWS_REGION=us-east-1 \
  AWS_ENDPOINT_URL=$GEN $s3c AWS_CONFIG_FILE=/dev/null AWS_SHARED_CREDENTIALS_FILE=/dev/null TF_PLUGIN_CACHE_DIR=/cache $RETRY "$tf" "$@" -no-color -input=false 2>&1; }
echo "config: generic=$GEN s3control=${S3C:-<unset>} hosts-entry=$HOSTS offline=$OFFLINE retry=${RETRY:-default}"
run init >/dev/null
[ "$HOSTS" = 1 ] && echo "127.0.0.1 000000000041.localhost" >> /etc/hosts
[ "$OFFLINE" = 1 ] && printf "nameserver 192.0.2.1\noptions timeout:1 attempts:1\n" > /etc/resolv.conf
st=$(date +%s); timeout ${APPLY_TO:-400} env -i PATH=/usr/bin:/bin HOME=/home/t AWS_ACCESS_KEY_ID=000000000041 AWS_SECRET_ACCESS_KEY=test AWS_REGION=us-east-1 AWS_ENDPOINT_URL=$GEN $s3c AWS_CONFIG_FILE=/dev/null AWS_SHARED_CREDENTIALS_FILE=/dev/null TF_PLUGIN_CACHE_DIR=/cache $RETRY "$tf" apply -auto-approve -no-color -input=false > /tmp/apply.log 2>&1; rc=$?
echo "apply rc=$rc elapsed=$(( $(date +%s) - st ))s: $(grep -E "^Error|Apply complete" /tmp/apply.log | head -2 | cut -c1-330 | tr "\n" " ")"
grep -oE "Get \"http://[^/]+/" /tmp/apply.log | sort -u | head -3
st=$(date +%s); timeout ${DESTROY_TO:-300} env -i PATH=/usr/bin:/bin HOME=/home/t AWS_ACCESS_KEY_ID=000000000041 AWS_SECRET_ACCESS_KEY=test AWS_REGION=us-east-1 AWS_ENDPOINT_URL=$GEN $s3c AWS_CONFIG_FILE=/dev/null AWS_SHARED_CREDENTIALS_FILE=/dev/null TF_PLUGIN_CACHE_DIR=/cache $RETRY "$tf" destroy -auto-approve -no-color -input=false > /tmp/destroy.log 2>&1; rc=$?
echo "destroy rc=$rc elapsed=$(( $(date +%s) - st ))s: $(grep -E "^Error|Destroy complete" /tmp/destroy.log | head -2 | cut -c1-330 | tr "\n" " ")"
' 2>&1
