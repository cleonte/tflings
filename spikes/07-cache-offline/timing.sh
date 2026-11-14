#!/bin/bash
# usage: timing.sh <1.12|latest>  — native macOS, warm plugin cache, Floci on localhost:4577 (default env.sh endpoints), account 000000000071
v=$1; here=$(cd "$(dirname "$0")" && pwd); acct=000000000071
export ENDPOINT=http://127.0.0.1:4577
run() { ( cd "$w" && ACCOUNT=$acct "$here/../env.sh" env AWS_ENDPOINT_URL_S3_CONTROL=http://localhost:4577 "$here/../.cache/bin/terraform-$v" "$@" >/dev/null 2>&1 ); }
runerr() { ( cd "$w" && ACCOUNT=$acct "$here/../env.sh" env AWS_ENDPOINT_URL_S3_CONTROL=http://localhost:4577 "$here/../.cache/bin/terraform-$v" "$@" 2>&1 ); }
med() { # command... ; 5 runs, prints median seconds
  local ts=(); for i in 1 2 3 4 5; do s=$(python3 -c 'import time;print(time.time())'); "$@"; ts+=($(python3 -c "import time;print(round(time.time()-$s,2))")); done
  printf '%s\n' "${ts[@]}" | sort -n | sed -n 3p; }
for cfg in empty one thirteen; do
  :
  w=$here/work-$v-$cfg; rm -rf "$w"; mkdir -p "$w"; cp "$here/$cfg.tf" "$w/main.tf"
  run init -no-color -input=false; [ $cfg != empty ] && run apply -auto-approve -no-color -input=false
  rc=$(runerr plan -out=tfplan -no-color -input=false >/dev/null; echo $?); rv=$(runerr validate -no-color >/dev/null; echo $?); rs=$(runerr show -json -no-color tfplan >/dev/null; echo $?); echo "(exit codes: plan -out $rc, validate $rv, show -json $rs)"
  echo "== $v config=$cfg: validate $(med run validate -no-color)s | plan $(med run plan -no-color -input=false)s | plan -refresh=false $(med run plan -refresh=false -no-color -input=false)s | plan -out $(med run plan -out=tfplan -no-color -input=false)s | show -json plan $(med run show -json -no-color tfplan)s | per-save (validate+plan -out+show -json) $(med bash -c "$(declare -f run); w=$w; v=$v; here=$here; acct=$acct; export ENDPOINT; run validate -no-color; run plan -out=tfplan -no-color -input=false; run show -json -no-color tfplan")s"
  [ $cfg != empty ] && run destroy -auto-approve -no-color -input=false
done
