#!/bin/bash
# usage: run.sh <1.12|latest> <terraform args...>   (runs in a per-version copy so state/.terraform do not mix)
set -u
v=$1; shift
here=$(cd "$(dirname "$0")" && pwd)
w=$here/work-$v
mkdir -p "$w" && cp "$here/main.tf" "$w/"
cd "$w" && exec "$here/../env.sh" "$here/../.cache/bin/terraform-$v" "$@"
