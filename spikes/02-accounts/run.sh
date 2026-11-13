#!/bin/bash
# usage: run.sh <account-id> <terraform args...>   (Terraform latest, own work dir + state per account)
a=$1; shift; here=$(cd "$(dirname "$0")" && pwd); w=$here/work-$a
mkdir -p "$w" && cp "$here/main.tf" "$w/" && cd "$w" && ACCOUNT=$a exec "$here/../env.sh" "$here/../.cache/bin/terraform-latest" "$@" -no-color -input=false
