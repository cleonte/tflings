#!/bin/sh
# Floci S3 Control TagResource replaces the tag set instead of adding to it.
# Talks only to a local Floci (default 127.0.0.1:4577, tflings's), with a fake
# account id as the key. Usage: repro.sh [port]
set -eu
PORT=${1:-4577}
A=535353535353
B=tag-merge-repro-$$
AUTH="Authorization: AWS4-HMAC-SHA256 Credential=$A/20261115/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=00"
ARN="arn%3Aaws%3As3%3A%3A%3A$B"
# <account>.localhost is what the AWS SDK builds for S3 Control; it resolves to loopback on most systems.
URL="http://$A.localhost:$PORT/v20180820/tags/$ARN"

tag() {
  curl -s -o /dev/null -w "TagResource  -> HTTP %{http_code}\n" -X POST -H "$AUTH" -H "x-amz-account-id: $A" \
    -H "Content-Type: application/xml" \
    --data "<TagResourceRequest xmlns=\"http://awss3control.amazonaws.com/doc/2018-08-20/\"><Tags>$1</Tags></TagResourceRequest>" "$URL"
}
list() {
  echo "ListTagsForResource -> $(curl -s -H "$AUTH" -H "x-amz-account-id: $A" "$URL" | grep -o '<Key>[^<]*</Key><Value>[^<]*</Value>' | sed 's/<Key>//;s/<\/Key><Value>/=/;s/<\/Value>//' | tr '\n' ' ')"
}

curl -s -o /dev/null -w "CreateBucket  -> HTTP %{http_code}\n" -X PUT -H "$AUTH" "http://127.0.0.1:$PORT/$B"
echo "== tag a=1"
tag '<Tag><Key>a</Key><Value>1</Value></Tag>'
list
echo "== tag b=2 and c=3 (AWS adds them: a, b and c)"
tag '<Tag><Key>b</Key><Value>2</Value></Tag><Tag><Key>c</Key><Value>3</Value></Tag>'
list
curl -s -o /dev/null -X DELETE -H "$AUTH" "http://127.0.0.1:$PORT/$B"
