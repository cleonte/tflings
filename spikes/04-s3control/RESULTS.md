# Spike 04 — S3 Control endpoint and the account-ID host prefix

Run 2026-11-13. Floci 2.4.0 in Docker, Terraform 1.17.2 in Linux containers (Debian 13, glibc, linux/arm64), plus 1.12.2 and 1.17.2 natively on macOS arm64; hashicorp/aws 6.73.0. Config: spike 01's 13 resources plus `force_destroy = true` on the bucket and `depends_on` ordering of the S3 object after the versioning resource. Account `000000000041`; Floci reset (`/_floci/state/reset`) before every run so orphans from a failed run cannot pollute the next. "Offline" = after `init`, `/etc/resolv.conf` replaced by an unreachable nameserver (`192.0.2.1`).

The task text for this spike was cut off after "If" in the message I received, so I ran the matrix below and stopped; whatever the "If ..." branch asked for is not done.

## Question: is `AWS_ENDPOINT_URL_S3_CONTROL` used verbatim, without the account-ID prefix?

**No.** The variable is honoured (it replaces the endpoint for S3 Control only), but the SDK still prepends `<account-id>.` to its host. With an IP address the result cannot resolve:

```
generic=http://127.0.0.1:4577  s3control=http://127.0.0.1:4577
Error: listing tags for S3 Bucket (spike01-data-000000000041): operation error S3 Control: ListTagsForResource ... Get "http://000000000041.127.0.0.1:4577/v20180820/tags/...": dial tcp: lookup ... no such host
```

Same on macOS (spike 01) and Debian. So no address that needs no DNS can be given for S3 Control; some `<account>.<host>` name must resolve.

## What the variable is good for: split the endpoints

Because the generic endpoint can be an IP (S3 data calls then use path style and need no DNS; spike 03 showed the same for the S3 backend), only the S3 Control host needs a name:

`AWS_ENDPOINT_URL=http://127.0.0.1:4577` + `AWS_ENDPOINT_URL_S3_CONTROL=http://localhost:4577`

| # | Generic | S3 Control | DNS | `/etc/hosts` line | Result |
|---|---|---|---|---|---|
| T1 | `127.0.0.1` | `127.0.0.1` | online | no | **FAIL** (`000000000041.127.0.0.1`), `destroy` fails the same way (8 s / 2 s) |
| T2 | `127.0.0.1` | `localhost` | online | no | **PASS** apply 13 added (8 s), destroy 13 (4 s) |
| T3 | `127.0.0.1` | `localhost` | **offline** | no | **FAIL**: apply still running at 90 s (killed by `timeout`; `AWS_MAX_ATTEMPTS=1 AWS_RETRY_MODE=standard` set) |
| T4 | `127.0.0.1` | `localhost` | **offline** | `127.0.0.1 000000000041.localhost` | **PASS** apply 13 added (8 s), destroy 13 (3 s) |
| T5 | `localhost` (the spike 01 default) | unset | **offline** | `127.0.0.1 000000000041.localhost` | **FAIL**: apply still running at 90 s (the bucket host `<bucket>.localhost` is also needed) |
| macOS | `127.0.0.1` | `localhost` | online | no | **PASS** on Terraform 1.12.2 and 1.17.2: apply 13, plan empty, destroy 13 |

## Conclusions

1. **Best configuration found: generic `http://127.0.0.1:4577`, S3 Control `http://localhost:4577`.** It cuts the DNS dependency from "one name per account plus one per bucket" (spike 01/03) to exactly **one name per account: `<account-id>.localhost`**, used only for S3 Control (bucket tags). All other calls, including the S3 state backend, use the IP.
2. **Offline still needs that one name.** With no DNS, it works only if `<account-id>.localhost` is resolvable (T4: one line in `/etc/hosts`; needs root, no wildcard). Because tflings derives the account ID from the exercise name, the set of IDs is known in advance, so `doctor` could print the exact lines, but a learner has to add them with `sudo`. If account IDs are many (one per exercise and reset), that list is long and changes at every reset. Not solved.
3. **`AWS_MAX_ATTEMPTS` does not bound the offline failure.** With `AWS_MAX_ATTEMPTS=1` the provider still retried for the full 90 s (T3, T5), and the S3 backend still made 5 attempts (spike 03). The decision "runner sets `AWS_MAX_ATTEMPTS` low" does not give a fast, clear error for this failure; a runner-side timeout (kill after N seconds and print a `doctor`-style message) is needed. Not investigated why (provider-level waiters are a guess).
4. A destroy after an interrupted apply worked (T3/T5 destroy: 8 resources, 2 s), so killing an offline-hung apply does not leave the exercise unrecoverable.

## Not tested

- Windows, macOS offline, Linux amd64, systemd-resolved, Alpine/musl for this matrix.
- Whether some other DNS-free route exists for S3 Control (for example avoiding the call). The tags call is made even for a bucket with no tags.
- Options needing no root (user/mount namespaces with a private `/etc/hosts`, a resolver embedded in tflings) — ideas only.

## Proposed design change (for review; `ROADMAP.md` not edited for this)

Runner env: `AWS_ENDPOINT_URL=http://127.0.0.1:4577`, `AWS_ENDPOINT_URL_S3_CONTROL=http://localhost:4577`; a wall-clock timeout around every Terraform call; `doctor` resolves `<account>.localhost` and prints the `/etc/hosts` lines when it fails. This replaces "endpoint is always `http://localhost:4577`" in `CLAUDE.md` and the ROADMAP.

## Files

`main.tf`, `linux-s3control.sh` (one Linux container run; env: `GEN`, `S3C`, `HOSTS`, `OFFLINE`, `RETRY`, `APPLY_TO`, `DESTROY_TO`), `matrix.sh` (T3–T5), `matrix.out`, `mac.sh`.
