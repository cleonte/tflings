#!/bin/bash
# Does <account>.localhost resolve for the AWS provider on Linux?
# usage: linux-check.sh <image> <1.12|latest> <host|shared> [nodns]
#   nodns  : after `init`, /etc/resolv.conf is replaced by an unreachable nameserver (192.0.2.1), i.e. an offline learner
#   host   : --network host (inside the Lima VM, Floci is published on its loopback :4577) — mirrors a Linux learner with terraform on the host
#   shared : --network container:tflings-floci (same network namespace as Floci; Floci listens on :4566 there)
img=$1; v=$2; mode=$3; nodns=0; [ "${4:-}" = nodns ] && nodns=1
spikes=$(cd "$(dirname "$0")/.." && pwd)
if [ "$mode" = host ]; then net=host; ep=http://localhost:4577; else net=container:tflings-floci; ep=http://localhost:4566; fi
docker run --rm --user root --network $net -v "$spikes":/spikes:ro -v tflings-linux-plugins:/cache -e EP=$ep -e V=$v -e NODNS=$nodns "$img" sh -c '
set -u
. /etc/os-release; echo "image: $PRETTY_NAME  libc: $(ldd --version 2>&1 | head -1)"
if command -v apt-get >/dev/null; then apt-get update -qq >/dev/null 2>&1; apt-get install -y -qq ca-certificates >/dev/null 2>&1
elif command -v apk >/dev/null; then apk add -q --no-cache ca-certificates >/dev/null 2>&1; fi
mkdir -p /work /home/t && cp /spikes/01-env-only/main.tf /work/ && cd /work
tf=/spikes/.cache/bin/terraform-linux-$V
run() { env -i PATH=/usr/bin:/bin HOME=/home/t AWS_ACCESS_KEY_ID=000000000099 AWS_SECRET_ACCESS_KEY=test AWS_REGION=us-east-1 \
  AWS_ENDPOINT_URL=$EP AWS_CONFIG_FILE=/dev/null AWS_SHARED_CREDENTIALS_FILE=/dev/null TF_PLUGIN_CACHE_DIR=/cache "$tf" "$@" -no-color -input=false 2>&1; }
echo "terraform: $("$tf" version | head -1)"
run init >/dev/null; echo "init rc=$?"
[ "$NODNS" = 1 ] && printf "nameserver 192.0.2.1\noptions timeout:1 attempts:1\n" > /etc/resolv.conf && echo "resolv.conf: unreachable nameserver only"
echo "getent 000000000001.localhost: $(getent hosts 000000000001.localhost || echo FAIL)"
run apply -auto-approve | grep -E "^Error|Apply complete|dial tcp" | head -4
run plan -detailed-exitcode | grep -E "^Error|No changes|Plan:|dial tcp" | head -3
run destroy -auto-approve | grep -E "^Error|Destroy complete|dial tcp" | head -3
' 2>&1
