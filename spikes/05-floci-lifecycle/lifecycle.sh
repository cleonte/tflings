#!/bin/bash
# Floci container lifecycle measurements. Own container/volume/port (4578) so the working tflings-floci is untouched.
IMG=floci/floci@sha256:5ee85db604c4aa74ac76e07ea5cc83086f37bb472bf3dbb97e34019e48efae9c
N=tflings-floci-spike05; V=tflings-floci-spike05-data; P=4578
now() { python3 -c 'import time;print(time.time())'; }
wait_health() { # $1 = url ; prints seconds until HTTP 200
  local s=$(now); until curl -fs -o /dev/null "http://localhost:$P/_floci/health"; do sleep 0.02; done; python3 -c "import time;print(f'{time.time()-$s:.2f}')"; }
run() { docker run -d --name $N -p 127.0.0.1:$P:4566 -e FLOCI_STORAGE_MODE=persistent -e FLOCI_STORAGE_PERSISTENT_PATH=/data -v $V:/data "$@" $IMG >/dev/null; }
median() { sort -n | awk '{a[NR]=$1} END{print a[int((NR+1)/2)]}'; }
cleanup() { docker rm -f $N >/dev/null 2>&1; }
cleanup; docker volume rm $V >/dev/null 2>&1
echo "== cold start (docker run -> /_floci/health 200), empty volume then reused volume, 5 runs each"
for vol in empty reused; do
  ts=()
  for i in 1 2 3 4 5; do cleanup; [ $vol = empty ] && docker volume rm $V >/dev/null 2>&1; s=$(now); run; t=$(wait_health); ts+=($t); done
  echo "$vol volume: ${ts[*]}  median=$(printf '%s\n' "${ts[@]}" | median)s"
done
echo "== docker healthcheck status after start"; sleep 6; docker inspect $N --format '{{.State.Health.Status}}'
echo "== idle memory"; docker stats --no-stream --format 'mem={{.MemUsage}} cpu={{.CPUPerc}}' $N
echo "== docker stop / start / restart timings"
s=$(now); docker stop $N >/dev/null; python3 -c "import time;print(f'stop: {time.time()-$s:.2f}s')"
s=$(now); docker start $N >/dev/null; echo "start->healthy: $(wait_health)s"
s=$(now); docker restart $N >/dev/null; echo "restart->healthy: $(wait_health)s"
echo "== persistence across container removal (same volume)"
AWS="docker run --rm --network host -e AWS_ACCESS_KEY_ID=000000000051 -e AWS_SECRET_ACCESS_KEY=test -e AWS_REGION=us-east-1 -e AWS_ENDPOINT_URL=http://localhost:$P -e AWS_CONFIG_FILE=/dev/null -e AWS_SHARED_CREDENTIALS_FILE=/dev/null -e AWS_PAGER= amazon/aws-cli"
$AWS ssm put-parameter --name /persist --type String --value kept --query Version --output text
docker rm -f $N >/dev/null; run; wait_health >/dev/null
echo "after rm + new container: $($AWS ssm get-parameter --name /persist --query Parameter.Value --output text 2>&1 | tail -1)"
echo "== port already in use (second container, same port)"
docker run -d --name ${N}-b -p 127.0.0.1:$P:4566 $IMG 2>&1 | tail -1; docker rm -f ${N}-b >/dev/null 2>&1
echo "== name already in use"; docker run -d --name $N $IMG 2>&1 | tail -1
echo "== docker daemon unreachable"; DOCKER_HOST=unix:///nonexistent.sock docker info 2>&1 | head -2 | cut -c1-200
echo "== hardened run (cap-drop ALL, no-new-privileges, read-only root, tmpfs /tmp)"
cleanup; run --cap-drop ALL --security-opt no-new-privileges --read-only --tmpfs /tmp; echo "healthy in $(timeout 30 bash -c "$(declare -f now wait_health); P=$P; wait_health" 2>&1)s"; $AWS sts get-caller-identity --query Account --output text 2>&1 | tail -1
echo "== no docker.sock mounted:"; docker inspect $N --format '{{range .Mounts}}{{.Type}}:{{.Source}}->{{.Destination}} {{end}}'
cleanup; docker volume rm $V >/dev/null 2>&1; echo done
