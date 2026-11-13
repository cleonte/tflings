#!/bin/bash
D=mcr.microsoft.com/devcontainers/base:trixie; R="AWS_MAX_ATTEMPTS=1 AWS_RETRY_MODE=standard"
t() { curl -s -X POST localhost:4577/_floci/state/reset >/dev/null; echo "##### $1"; shift; env APPLY_TO=90 DESTROY_TO=60 "$@" ./linux-s3control.sh $D latest; }
t "T3 OFFLINE, generic=127.0.0.1, s3control=localhost, retry capped" GEN=http://127.0.0.1:4577 S3C=http://localhost:4577 OFFLINE=1 "RETRY=$R"
t "T4 OFFLINE + /etc/hosts line for <account>.localhost, generic=127.0.0.1, s3control=localhost" GEN=http://127.0.0.1:4577 S3C=http://localhost:4577 OFFLINE=1 HOSTS=1 "RETRY=$R"
t "T5 OFFLINE + /etc/hosts line, generic=localhost (default, no s3control var) -> bucket.localhost also needed" GEN=http://localhost:4577 OFFLINE=1 HOSTS=1 "RETRY=$R"
