# Phase 0 spikes — summary

2026-11-13. macOS arm64, Floci 2.4.0 in Docker (Lima), Terraform 1.12.2 and 1.17.2, hashicorp/aws 6.73.0. Details and commands are in each spike's `RESULTS.md`. The resulting `ROADMAP.md` and `CLAUDE.md` changes were applied in commit `26d44d9`.

| Spike | Result | Decision / consequence |
|---|---|---|
| 01 env-only | PASS (with a change) | Provider block is only `region`; no `skip_*`, no `endpoints {}`. Warm `plan` median 2.9 s (13 resources), cold `init` 11 s, provider cache 737 MB. Secrets need `recovery_window_in_days = 0`. Endpoint must not be a bare IP for S3 Control (see 04). |
| 00 Linux resolution | PARTIAL | `<account>.localhost` resolves on Debian (glibc) and Alpine (musl) when DNS is available. With no DNS it fails and `apply` hangs. Windows, macOS offline, amd64, systemd-resolved untested. |
| 02 accounts | PASS | One account per exercise via 12-digit key ID; full isolation; same names allowed in two accounts (per-account S3 namespace). No single-account reset API: fresh account ID per reset; scripted wipe 8.8 s as fallback; global `/_floci/state/reset` 0.05 s. |
| 03 S3 backend | PASS | `use_lockfile` and `dynamodb_table` work on both versions with `bucket`/`key`/`region` only. Stuck lock + `force-unlock` and `init -migrate-state` work. `dynamodb_table` warns as deprecated on 1.17.2, not on 1.12.2. With a `localhost` endpoint the backend needs `<bucket>.localhost`; with `127.0.0.1` it needs no DNS. |
| 04 S3 Control | PASS (online) / OPEN (offline) | `AWS_ENDPOINT_URL_S3_CONTROL` is honoured but the `<account-id>.` prefix stays. Best setup: generic `127.0.0.1` + S3 Control `localhost`, leaving one name per account to resolve. Offline Linux needs `127.0.0.1 <account>.localhost` in `/etc/hosts` (root). `AWS_MAX_ATTEMPTS=1` does not stop the offline hang. |

## Open problems

1. **Offline learners.** One `/etc/hosts` line per account ID, with root, and a fresh account ID per reset makes that list grow. No root-free fix tested.
2. **No fast failure.** `AWS_MAX_ATTEMPTS` is not enough; the runner needs a wall-clock timeout and a clear message.
3. **Untested:** Windows, macOS offline, Linux amd64, systemd-resolved, the Floci native binary (2.4.0 published none), whether `terraform_data`/`random` lessons need Docker at all.
4. **Unexplained:** one provider-startup failure on two back-to-back runs (spike 03); one `BucketNotEmpty` on destroy (spike 00), cause not verified.
