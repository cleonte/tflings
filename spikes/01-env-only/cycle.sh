#!/bin/bash
# usage: cycle.sh <1.12|latest>  — init, plan, apply, 5x no-change plan (timed), destroy
v=$1; cd "$(dirname "$0")"
t() { local s=$(python3 -c 'import time;print(time.time())'); "$@" >/tmp/cycle.out 2>&1; local rc=$?; python3 -c "import time;print(f'{time.time()-$s:.2f}s rc=$rc')"; return 0; }
echo "terraform: $(./run.sh $v version | head -1)"
./run.sh $v init -no-color -input=false >/dev/null 2>&1
echo -n "apply: "; t ./run.sh $v apply -auto-approve -no-color -input=false; grep -E 'Apply complete|Error' /tmp/cycle.out
echo "plan after apply (detailed-exitcode: 0=empty, 2=diff):"
for i in 1 2 3 4 5; do echo -n " run $i: "; t ./run.sh $v plan -detailed-exitcode -no-color -input=false; done
./run.sh $v plan -no-color -input=false 2>&1 | grep -E 'No changes|Plan:|must be replaced|will be updated' | head
echo -n "destroy: "; t ./run.sh $v destroy -auto-approve -no-color -input=false; grep -E 'Destroy complete|Error' /tmp/cycle.out
