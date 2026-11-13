#!/bin/bash
# usage: stuck.sh <1.12|latest> <lockfile|ddb>
v=$1; m=$2; d=$v-$m; cli=../02-accounts/awscli.sh; A=000000000031
echo "=== $v $m: start apply holding the lock, then kill -9"
./run.sh $v $d apply -auto-approve -no-color -input=false -var id=$d -var hold=60 -replace=terraform_data.hold >/tmp/stuck-$d.log 2>&1 &
pid=$!
for i in $(seq 1 40); do
  if [ $m = lockfile ]; then $cli $A s3api head-object --bucket tfstate-spike03 --key spike03/$d.tfstate.tflock >/dev/null 2>&1 && break
  else [ "$($cli $A dynamodb scan --table-name tflock-spike03 --query 'Count' --output text 2>/dev/null)" -gt 0 ] 2>/dev/null && break; fi
  sleep 1
done
echo "lock object/item while running:"
if [ $m = lockfile ]; then $cli $A s3api list-objects-v2 --bucket tfstate-spike03 --query 'Contents[].Key' --output text | tr '\t' '\n' | grep "$d"
else $cli $A dynamodb scan --table-name tflock-spike03 --query 'Items[].LockID.S' --output text; fi
kill -9 $pid; pkill -9 -f 'sleep 60'; sleep 1
echo "--- second run (plan) while the lock is stuck:"
./run.sh $v $d plan -no-color -input=false -var id=$d 2>&1 | sed -n '/Error/,$p' | head -16 | tee /tmp/stuck-err-$d.txt
id=$(grep -E '^\s*ID:' /tmp/stuck-err-$d.txt | awk '{print $2}')
echo "--- lock id parsed: $id"
echo "--- force-unlock:"; ./run.sh $v $d force-unlock -force -no-color $id 2>&1 | head -3
echo "--- plan after unlock:"; ./run.sh $v $d plan -detailed-exitcode -no-color -input=false -var id=$d 2>&1 | grep -E 'No changes|^Plan|^Error'
