#!/bin/bash
# macOS native: generic endpoint 127.0.0.1 (path-style S3) + S3 Control on localhost. usage: mac.sh <1.12|latest>
v=$1; here=$(cd "$(dirname "$0")" && pwd); w=$here/work-mac-$v; mkdir -p $w; cp $here/main.tf $w/; cd $w
r() { ACCOUNT=000000000041 ENDPOINT=http://127.0.0.1:4577 ../../env.sh env AWS_ENDPOINT_URL_S3_CONTROL=http://localhost:4577 "$here/../.cache/bin/terraform-$v" "$@" -no-color -input=false 2>&1; }
r init >/dev/null; echo "== $v apply: $(r apply -auto-approve | grep -E '^Error|Apply complete' | cut -c1-200)"
echo "== $v plan: $(r plan -detailed-exitcode | grep -E 'No changes|^Plan|^Error' | cut -c1-200)"
echo "== $v destroy: $(r destroy -auto-approve | grep -E '^Error|Destroy complete' | cut -c1-200)"
