#!/bin/bash
# usage: awscli.sh <account-id> <aws args...>  — real AWS CLI v2 in a container, clean env, Floci only (host network inside the Lima VM)
a=$1; shift
exec docker run --rm --network host -e AWS_ACCESS_KEY_ID=$a -e AWS_SECRET_ACCESS_KEY=test -e AWS_REGION=us-east-1 \
  -e AWS_ENDPOINT_URL=http://localhost:4577 -e AWS_CONFIG_FILE=/dev/null -e AWS_SHARED_CREDENTIALS_FILE=/dev/null -e AWS_PAGER= \
  amazon/aws-cli "$@"
