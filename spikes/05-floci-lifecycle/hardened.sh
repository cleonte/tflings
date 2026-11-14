#!/bin/bash
# Hardened run: dropped capabilities, no-new-privileges, read-only root; persistent state on a named volume at /app/data.
IMG=floci/floci@sha256:5ee85db604c4aa74ac76e07ea5cc83086f37bb472bf3dbb97e34019e48efae9c
H=(--cap-drop ALL --cap-add SYS_CHROOT --cap-add SETUID --cap-add SETGID --security-opt no-new-privileges --read-only --tmpfs /tmp)
aws() { docker run --rm --network host -e AWS_ACCESS_KEY_ID=000000000052 -e AWS_SECRET_ACCESS_KEY=test -e AWS_REGION=us-east-1 -e AWS_ENDPOINT_URL=http://localhost:4578 -e AWS_CONFIG_FILE=/dev/null -e AWS_SHARED_CREDENTIALS_FILE=/dev/null -e AWS_PAGER= amazon/aws-cli "$@"; }
run() { docker run -d --name s05 -p 127.0.0.1:4578:4566 -e FLOCI_STORAGE_MODE=persistent -e FLOCI_STORAGE_PERSISTENT_PATH=/app/data -v tflings-s05-data:/app/data "${H[@]}" $IMG >/dev/null
  for i in $(seq 1 50); do curl -fs -o /dev/null localhost:4578/_floci/health && return; sleep 0.1; done; echo UNHEALTHY; docker logs s05 | tail -3; }
docker rm -f s05 >/dev/null 2>&1; docker volume rm tflings-s05-data >/dev/null 2>&1
run; aws ssm put-parameter --name /p --type String --value hardened --query Version --output text; aws s3api create-bucket --bucket b52 --query Location --output text
docker rm -f s05 >/dev/null; run
echo "after recreate: param=$(aws ssm get-parameter --name /p --query Parameter.Value --output text) buckets=$(aws s3api list-buckets --query 'length(Buckets)' --output text)"
docker rm -f s05 >/dev/null; docker volume rm tflings-s05-data >/dev/null
