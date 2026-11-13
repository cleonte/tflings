#!/bin/sh
# Floci 2.4.0 in Docker (no native binary is published for 2.4.0), 127.0.0.1:4577, data in the named Docker volume tflings-floci-data
# (Lima mounts $HOME read-only, so a bind mount under spikes/.cache cannot be written).
here=$(cd "$(dirname "$0")" && pwd)
name=tflings-floci
case "$1" in
  up)
    docker rm -f $name >/dev/null 2>&1
    docker run -d --name $name -p 127.0.0.1:4577:4566 \
      -e FLOCI_STORAGE_MODE=persistent -e FLOCI_STORAGE_PERSISTENT_PATH=/data \
      -v tflings-floci-data:/data floci/floci:2.4.0 >/dev/null || exit 1
    for i in $(seq 1 60); do
      curl -fs http://127.0.0.1:4577/_floci/health >/dev/null && { echo up; exit 0; }
      sleep 0.5
    done
    echo "floci did not become healthy" >&2; docker logs $name | tail -20; exit 1 ;;
  down) docker rm -f $name >/dev/null 2>&1; echo down ;;
  wipe) docker rm -f $name >/dev/null 2>&1; docker volume rm tflings-floci-data >/dev/null 2>&1; echo wiped ;;
  status) curl -s http://127.0.0.1:4577/_floci/health || echo "not running" ;;
  *) echo "usage: $0 up|down|wipe|status" >&2; exit 2 ;;
esac
