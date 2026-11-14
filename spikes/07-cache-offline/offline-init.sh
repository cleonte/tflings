#!/bin/bash
# Offline `init` with the AWS provider, Linux container. Network "offline" = unreachable nameserver after the online prep step.
# usage: offline-init.sh <image> <1.12|latest>
img=$1; v=$2; spikes=$(cd "$(dirname "$0")/.." && pwd)
docker run --rm --user root -v "$spikes":/spikes:ro -v tflings-linux-plugins:/cache -e V=$v "$img" sh -c '
set -u
if command -v apt-get >/dev/null; then apt-get update -qq >/dev/null 2>&1; apt-get install -y -qq ca-certificates >/dev/null 2>&1; fi
tf=/spikes/.cache/bin/terraform-linux-$V; mkdir -p /home/t
tfr() { env -i PATH=/usr/bin:/bin HOME=/home/t TF_PLUGIN_CACHE_DIR=/cache $X "$tf" "$@" -no-color -input=false 2>&1; }
mkdir -p /prep && cp /spikes/07-cache-offline/empty.tf /prep/main.tf && (cd /prep && X="" tfr init >/dev/null)
cp /prep/.terraform.lock.hcl /full.lock; grep -v "\"h1:" /full.lock > /zhonly.lock
echo "lock file written online: $(grep -c "h1:" /full.lock) h1 hash, $(grep -c "zh:" /full.lock) zh hashes; zh-only copy: $(grep -c "h1:" /zhonly.lock) h1"
printf "nameserver 192.0.2.1\noptions timeout:1 attempts:1\n" > /etc/resolv.conf
case_() { # name lockfile cachedir extra-env
  local name=$1 lock=$2 cache=$3; shift 3; rm -rf /w && mkdir /w && cp /spikes/07-cache-offline/empty.tf /w/main.tf; [ -n "$lock" ] && cp $lock /w/.terraform.lock.hcl
  echo "== $name"; cd /w
  local out; local s=$(date +%s%N)
  out=$(env -i PATH=/usr/bin:/bin HOME=/home/t TF_PLUGIN_CACHE_DIR=$cache "$@" "$tf" init $INITARGS -no-color -input=false 2>&1); local rc=$?
  echo "  rc=$rc $(( ($(date +%s%N) - s) / 1000000 )) ms"; echo "$out" | grep -E "^Error|Terraform has been successfully|checksums|could not|Failed|cached" | head -3 | cut -c1-260
  [ -d .terraform/providers ] && echo "  .terraform size: $(du -sh .terraform | cut -f1); provider file type: $(find .terraform/providers -name "terraform-provider-aws*" -exec ls -ld {} \; | cut -c1 | head -1) (l = symlink)"
  cd /
}
mkdir -p /emptycache; INITARGS=""
case_ "A cache filled, NO lock file, offline" "" /cache env
case_ "B cache filled, full lock (h1+zh), offline" /full.lock /cache env
case_ "C cache filled, zh-only lock, offline" /zhonly.lock /cache env
case_ "D = C + TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE=true" /zhonly.lock /cache env TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE=true
case_ "E EMPTY cache, full lock, offline" /full.lock /emptycache env
printf "provider_installation {\n  filesystem_mirror {\n    path = \"/cache\"\n  }\n}\n" > /mirror.tfrc
INITARGS="-plugin-dir=/cache" case_ "F init -plugin-dir=/cache, zh-only lock, no TF_PLUGIN_CACHE_DIR, offline" /zhonly.lock /emptycache env
INITARGS="-plugin-dir=/cache" case_ "F2 init -plugin-dir=/cache, NO lock file, offline" "" /emptycache env
INITARGS="" case_ "G CLI config filesystem_mirror=/cache, no lock file, offline" "" /emptycache env TF_CLI_CONFIG_FILE=/mirror.tfrc
INITARGS="" case_ "G2 filesystem_mirror + full lock, offline" /full.lock /emptycache env TF_CLI_CONFIG_FILE=/mirror.tfrc
INITARGS="" printf "provider_installation {\n  filesystem_mirror {\n    path    = \"/cache\"\n    include = [\"hashicorp/aws\"]\n  }\n  direct {\n    exclude = [\"hashicorp/aws\"]\n  }\n}\n" > /mirror-aws.tfrc
INITARGS="" case_ "H mirror for hashicorp/aws only, direct for the rest, zh-only lock, offline" /zhonly.lock /emptycache env TF_CLI_CONFIG_FILE=/mirror-aws.tfrc
case_ "G3 filesystem_mirror + zh-only lock, offline" /zhonly.lock /emptycache env TF_CLI_CONFIG_FILE=/mirror.tfrc
' 2>&1
