#!/bin/bash
# usage: migrate.sh <1.12|latest> <lockfile|ddb> — local state -> S3 with init -migrate-state
v=$1; m=$2; d=migrate-$v-$m; here=$(cd "$(dirname "$0")" && pwd); w=$here/work-$d
rm -rf "$w"; mkdir -p "$w"; sed '/backend "s3" {}/d' "$here/app.tf" > "$w/app.tf"; cp "$here"/backend-*.hcl "$w/"
r() { ( cd "$w" && ACCOUNT=000000000031 "$here/../env.sh" "$here/../.cache/bin/terraform-$v" "$@" -no-color -input=false 2>&1 ); }
echo "=== $v $m"; r init >/dev/null; r apply -auto-approve -var id=$d | grep -E 'Apply complete|^Error'
echo "local state file: $(ls -l "$w/terraform.tfstate" | awk '{print $5}') bytes"
cp "$here/app.tf" "$w/app.tf"
r init -migrate-state -force-copy -backend-config=backend-$m.hcl -backend-config="key=spike03/$d.tfstate" | grep -E 'Successfully|migrat|^Error|Deprecated' | head -4
echo "in S3: $(../02-accounts/awscli.sh 000000000031 s3api head-object --bucket tfstate-spike03 --key spike03/$d.tfstate --query ContentLength --output text 2>&1 | tail -1) bytes"
r plan -detailed-exitcode -var id=$d | grep -E 'No changes|^Plan|^Error'
echo "local file after migrate: $(ls -l "$w/terraform.tfstate" | awk '{print $5}') bytes"
