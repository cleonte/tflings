#!/bin/bash
# usage: run.sh <1.12|latest> <dir-name> <terraform args...>   (account 000000000031; each dir has its own copy of the config + .terraform)
v=$1; d=$2; shift 2; here=$(cd "$(dirname "$0")" && pwd); w=$here/work-$d
mkdir -p "$w"; [ "$d" = bootstrap ] && cp "$here/bootstrap/main.tf" "$w/" || cp "$here/app.tf" "$here"/backend-*.hcl "$w/"
cd "$w" && ACCOUNT=${ACCOUNT:-000000000031} exec "$here/../env.sh" "$here/../.cache/bin/terraform-$v" "$@"
