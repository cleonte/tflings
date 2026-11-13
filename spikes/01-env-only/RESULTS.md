# Spike 01 — AWS provider configured only through the environment

Run 2026-11-13, macOS arm64 (Darwin 27.2.0), Docker via Lima (`lima-docker` context).

## Versions

| Thing | Version |
|---|---|
| Floci | 2.4.0, Docker image `floci/floci:2.4.0` (digest `sha256:5ee85db604c4aa74ac76e07ea5cc83086f37bb472bf3dbb97e34019e48efae9c`, arm64), `FLOCI_STORAGE_MODE=persistent` |
| Terraform | 1.12.2 (latest 1.12 patch) and 1.17.2 (latest), SHA256 verified against `terraform_<v>_SHA256SUMS` |
| hashicorp/aws | 6.73.0 (constraint `~> 6.0`) |

## Result: PASS, with one required change

On both Terraform versions `init`, `plan`, `apply`, second `plan` and `destroy` work with **no** `skip_*` flag, no `endpoints {}` block and no `s3_use_path_style`. The provider block is only `region = "us-east-1"`. Provider credential validation (STS `GetCallerIdentity`) is answered by Floci, and `aws_caller_identity` returns the account taken from the access key (`000000000001`).

### Finding 1 (blocking, fixed from env): the endpoint must be `localhost`, not `127.0.0.1`

With `AWS_ENDPOINT_URL=http://127.0.0.1:4577`, `aws_s3_bucket` fails on both read paths with:

```
Error: listing tags for S3 (Simple Storage) Bucket (spike01-data-000000000001): operation error S3 Control: ListTagsForResource ... Get "http://000000000001.127.0.0.1:4577/v20180820/tags/...": dial tcp: lookup 000000000001.127.0.0.1: no such host
```

The S3 Control client prepends the account ID to the host, so the endpoint host has to be a name whose subdomains resolve. `000000000001.localhost` resolves to 127.0.0.1 on this macOS (checked with `getaddrinfo`); `000000000001.127.0.0.1` does not.

- Fix: `AWS_ENDPOINT_URL=http://localhost:4577`. Env only, no HCL. `spikes/env.sh` now defaults to it (override with `ENDPOINT=`).
- Tried and **did not** work: `AWS_ENDPOINT_URL_S3_CONTROL=http://127.0.0.1:4577` on top of the generic URL (the host prefix is still added).
- Not tried: `AWS_ENDPOINT_URL_S3` with path style. Not needed.
- Design impact: tflings must set the endpoint host to `localhost`, never an IP. On Linux and Windows `*.localhost` resolution is not guaranteed by the OS resolver (Go's resolver may not special-case it) — verify in CI before relying on it. Fallback hostnames to test: `localhost.localstack.cloud`-style wildcard DNS (needs internet, so undesirable) or a `/etc/hosts` entry (undesirable).

### Finding 2 (exercise design): Secrets Manager destroy is soft by default

`aws_secretsmanager_secret` without `recovery_window_in_days = 0`: after `destroy`, re-creating the same name fails with `InvalidRequestException: You can't perform this operation on the secret because it was marked for deletion.` (same as real AWS). Spike config sets `recovery_window_in_days = 0`. Every exercise that creates a secret must do this, or `reset` will break. The stuck secret was only removable by wiping Floci's volume.

### Finding 3: a failed create leaves the resource tainted

The first apply (before the endpoint fix) created the bucket, failed on the tags read, and Terraform tainted it; the next apply replaced it (`1 destroyed`). Expected Terraform behaviour, noted because the checker must never assume an interrupted apply leaves a clean state.

## Perpetual diffs

None. Second `plan -detailed-exitcode` exits 0 and prints "No changes" on both versions, for all 13 resources.

## Timings (this machine, Floci in Docker, 13 resources)

| | Terraform 1.12.2 | Terraform 1.17.2 |
|---|---|---|
| cold `init` (empty plugin cache, includes registry download) | 10.9 s | 10.6 s |
| warm `init` (plugin cache populated, fresh `.terraform`) | 1.28 s | 1.22 s |
| `apply` (13 resources) | 11.7 s | 11.8 s |
| `plan`, no changes, 5 runs | 2.85 2.91 2.87 2.85 2.87 → **median 2.87 s** | 2.93 2.98 2.94 2.94 2.91 → **median 2.94 s** |
| `destroy` | 3.4 s | 3.4 s |

Target "warm plan < 5 s": met (about 2.9 s). `aws_dynamodb_table` takes about 6 s of the apply by itself. Most of the plan time is starting the large provider binary and refreshing 13 resources; a smaller exercise will be faster (not measured here).

## Size

AWS provider 6.73.0 darwin_arm64: binary 764,426,784 bytes (729 MiB); plugin cache directory `du -sh` = 737 MB (one version, one platform). Both Terraform versions share the cache. Each learner machine needs ~0.75 GB per provider version, once; the `~> 6.0` pin keeps it to one version.

## Safety check: endpoint removed, fake keys

`spikes/01-env-only/safety.sh <version>`: same environment minus `AWS_ENDPOINT_URL`, fake keys (`000000000001` / `test`), `plan` then `apply -auto-approve`, fresh directory, empty state. Both versions:

```
Error: Retrieving AWS account details: validating provider credentials: retrieving caller identity from STS: operation error STS: GetCallerIdentity, https response error StatusCode: 403, ... api error InvalidClientTokenId: The security token included in the request is invalid.
```

Failure is at provider configuration, before any resource call. No state file was written. Nothing was created. Note: this check does send one `sts:GetCallerIdentity` request with an invalid key to the real AWS STS endpoint; it was run only with fake keys, as the brief says. Consequence for tflings: a missing `AWS_ENDPOINT_URL` degrades to "auth error", not to touching a real account, **as long as the runner never passes real credentials** (it builds the environment from scratch with `env -i`-style rules).

## Setup deviations (from the brief)

1. **Floci 2.4.0 has no native binary.** The release page lists no assets; `releases/download/2.4.0/floci-aws-darwin-arm64` and `sha256sums.txt` both return 404 (the installation doc describes the assets, but this release did not publish them). Fell back to Docker as the brief allows: `docker run -p 127.0.0.1:4577:4566 floci/floci:2.4.0`.
2. **Persistent storage is a named Docker volume, not a folder under `spikes/`.** Docker runs in Lima, which mounts `$HOME` read-only, so a bind mount under `spikes/.cache/` fails (`/data ... Read-only file system`, Floci refuses to start). `floci.sh` uses volume `tflings-floci-data`; `floci.sh wipe` removes it. Storage is still project-specific and not global. If a writable Lima mount is wanted instead, that is a change to the user's Lima config; not done.
3. `spikes/env.sh` takes an optional `ENDPOINT` override (default `http://localhost:4577`, see Finding 1).

## Files

- `main.tf` — the spike config (13 resources, two data sources).
- `run.sh <1.12|latest> <args>` — runs Terraform in `work-<version>/` through `../env.sh`.
- `cycle.sh <version>` — init, apply, 5 timed plans, destroy.
- `safety.sh <version>` — the endpoint-removed check.

## Questions for the user before spike 02

- `ROADMAP.md` (as read today) plans a built-in simulator (`tflings/tfl` provider) and says "no real AWS"; `CLAUDE.md` and `spikes/README.md` plan Floci with the real `hashicorp/aws` provider and refer to a "Floci integration" section that is not in `ROADMAP.md`. This spike proves the Floci route is viable; the roadmap needs to be reconciled in spike 04.
