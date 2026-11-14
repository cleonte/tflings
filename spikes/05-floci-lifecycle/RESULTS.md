# Spike 05 — tflings-managed Floci container (lifecycle)

Run 2026-11-14. macOS arm64, Docker Engine 29.9 in Lima (`lima-docker` context), linux/arm64 image. Image `floci/floci@sha256:5ee85db604c4aa74ac76e07ea5cc83086f37bb472bf3dbb97e34019e48efae9c` (tag 2.4.0, 367,976,431 bytes). Test container, volume and port (4578) are separate from the working `tflings-floci`. Scripts: `lifecycle.sh`, `hardened.sh`.

Not tested: native Linux Docker, Docker Desktop, Colima, OrbStack, Windows, amd64. This is one Docker setup.

## Result: PASS

| Measurement | Result |
|---|---|
| Pull by digest | the pinned digest resolves (`RepoDigests` shows it); cold pull of the sibling tag 2.3.0 (338 MB) took 6.0 s on this connection |
| `docker run` → `/_floci/health` 200 | median **0.07 s**, 5 runs on an empty volume (0.07–0.10) and 5 on a reused volume (0.06–0.07) |
| Built-in Docker `HEALTHCHECK` | present (`Interval 5s`, `StartPeriod 10s`, `StartInterval 50ms`, `Retries 5`); `docker inspect --format '{{.State.Health.Status}}'` reported `healthy` |
| Idle memory | 29 MiB, 0.02 % CPU |
| `docker stop` | 0.24 s |
| `docker start` → healthy | 0.17 s |
| `docker restart` → healthy | 0.12 s |
| State survives `docker rm -f` + new container on the same named volume | yes (SSM parameter read back; also an S3 bucket in the hardened run) |

Startup is fast enough that `run`/`watch` can start Floci on demand with no visible wait.

## Errors `tflings aws up` has to recognise (exact text)

- Port taken: `docker: Error response from daemon: failed to set up container networking: driver failed programming external connectivity on endpoint ...: Bind for 127.0.0.1:4578 failed: port is already allocated`
- Name taken: `Conflict. The container name "/s05-a" is already in use by container "...". You have to remove (or rename) that container to be able to reuse that name.`
- Daemon unreachable (`DOCKER_HOST` pointing at a missing socket): `failed to connect to the docker API at unix:///nonexistent.sock; check if the path is correct and if the daemon is running: dial unix /nonexistent.sock: connect: no such file or directory`

So `up` should inspect the existing container first (running and healthy: reuse; stopped: `start`; wrong image digest: recreate) instead of relying on these errors.

## Findings that change the container recipe

1. **Anonymous volume leak.** The image declares `VOLUME /app/data`. Every `docker run` that does not mount `/app/data` creates an anonymous volume, and `docker rm` without `-v` leaves it behind. My earlier recipe (`FLOCI_STORAGE_PERSISTENT_PATH=/data` with the named volume at `/data`) leaked one per `up`; the dangling-volume count on this Docker went from 21 to 26 during one series of five test containers. Fix, verified: put the named volume at `/app/data` and set `FLOCI_STORAGE_PERSISTENT_PATH=/app/data`; the container then has exactly one mount and creates no anonymous volume. `spikes/floci.sh` is updated this way (and uses `docker rm -fv`). The old dangling volumes on this machine are still there; I did not prune them because the Docker host also holds other projects' volumes.
2. **Hardened run works, with three capabilities added back.** `--cap-drop ALL` alone fails (`chroot: cannot change root directory to '/': Operation not permitted`), then `chroot: failed to set supplemental groups`. This works and passed health, an SSM write, an S3 bucket and persistence across container removal:
   `--cap-drop ALL --cap-add SYS_CHROOT --cap-add SETUID --cap-add SETGID --security-opt no-new-privileges --read-only --tmpfs /tmp` (with the named volume at `/app/data`). Inside, the process starts as root (`uid=0`) and the entrypoint drops privileges.
3. **No Docker socket is mounted** and none is needed for the services the course uses (S3, IAM, STS, VPC, SSM, Secrets Manager, DynamoDB). Container-backed Floci services (Lambda, ECS, EC2 `RunInstances`) would need the socket and are out of scope; the recipe should never mount it.
4. Persistent storage under Lima needs a named volume, not a bind mount (spike 01).

## Proposed recipe

```
docker run -d --name tflings-floci -p 127.0.0.1:4577:4566 \
  -e FLOCI_STORAGE_MODE=persistent -e FLOCI_STORAGE_PERSISTENT_PATH=/app/data \
  -v tflings-floci-data:/app/data \
  --cap-drop ALL --cap-add SYS_CHROOT --cap-add SETUID --cap-add SETGID \
  --security-opt no-new-privileges --read-only --tmpfs /tmp \
  floci/floci@sha256:5ee85db604c4aa74ac76e07ea5cc83086f37bb472bf3dbb97e34019e48efae9c
```

Wait on the container's health status (or `/_floci/health`). The hardened flags are a recommendation, not applied to `spikes/floci.sh`.
