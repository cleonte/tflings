# Spike 00 — does `<account>.localhost` resolve for the AWS provider on Linux?

Run 2026-11-13. Docker in Lima on macOS arm64, so every container is linux/arm64 (amd64 not tested). Floci 2.4.0 (`floci/floci:2.4.0`), Terraform 1.12.2 and 1.17.2 (linux_arm64 zips, SHA256 verified), hashicorp/aws 6.73.0 (`~> 6.0`).

Question: spike 01 needs `AWS_ENDPOINT_URL=http://localhost:4577` because S3 Control sends requests to `<account-id>.<endpoint host>`. On macOS `000000000001.localhost` resolves. Does it on Linux?

## Method

`spikes/00-setup/linux-check.sh <image> <1.12|latest> <host|shared> [nodns]` runs spike 01's `main.tf` (account `000000000099`) inside a container: `init`, `apply`, `plan -detailed-exitcode`, `destroy`, with the same `env -i` rules as `env.sh`. `plan` alone would not prove anything (an empty-state plan never calls S3 Control), so the check is apply + plan on existing state.

- `host`: `--network host`. Inside the Lima VM that is the loopback where Floci is published (`localhost:4577`). Mirrors a Linux learner running Terraform on the host.
- `shared`: `--network container:tflings-floci`, the same network namespace as Floci (Floci on `localhost:4566` there). This is the literal "same network as Floci" reading of the brief.
- I did not use the `hashicorp/terraform` image; the static Linux binary is mounted into a Debian and an Alpine container instead, so both libc families are covered.

## Results

| Image | libc | Terraform | Network | Result |
|---|---|---|---|---|
| Debian 13 (trixie) | glibc 2.41 | 1.12.2 | host | apply 13 added, plan empty, destroy OK |
| Debian 13 | glibc 2.41 | 1.17.2 | host | same |
| Debian 13 | glibc 2.41 | 1.17.2 | shared | same |
| Alpine 3.22 | musl | 1.12.2 | host | same |
| Alpine 3.22 | musl | 1.17.2 | shared | apply and plan OK; `destroy` failed with `BucketNotEmpty` on `aws_s3_bucket.data` (not a DNS error; see note) |

`getent hosts 000000000001.localhost` returned `::1` in both images, so S3 Control calls resolved with Docker's/Lima's DNS available. Floci answered over IPv4 (the provider fell back from `::1` to 127.0.0.1 without trouble).

## Result with no DNS (offline learner): FAIL

Same Debian container, host network; after `init` (which needs the network) `/etc/resolv.conf` was replaced with an unreachable nameserver (`192.0.2.1`, 1 s timeout):

```
getent 000000000001.localhost: FAIL
```

`terraform apply` then did not finish; I killed it after several minutes (it was retrying the failing S3 Control request). Alpine was not run in this mode.

Meaning: neither glibc 2.41 nor the Go provider synthesizes `*.localhost` locally. It resolves only when the configured resolver answers it. That is true for Docker Desktop/Lima DNS and, I expect but did not test, for systemd-resolved desktops and macOS. It is **not** guaranteed on a minimal Linux, WSL, a CI box with a restricted resolver, or an offline machine. For a tool whose selling point is "works offline", this is a real risk.

## Smallest fix: not found

I did not find or test a fix that keeps the endpoint in env only and works offline. Candidates, all **untested**:

1. Tell the learner to add `127.0.0.1 <account>.localhost` lines to `/etc/hosts` (needs root, one line per account ID; no wildcard in `/etc/hosts`). Acceptable only if tflings uses a small fixed set of account IDs.
2. A tiny wildcard DNS stub inside tflings (heavy, needs a port 53 or resolver config).
3. Run Terraform inside a container with Docker's DNS (breaks the "one static binary" idea).
4. `doctor` check: resolve `000000000001.localhost` and print the exact fix when it fails.
5. Avoid S3 Control entirely: not possible from env; the provider calls it for bucket tags.

Not tested: Windows, macOS with the network off, Linux amd64, systemd-resolved. Needs a decision before Phase 1.

## Note on the one destroy failure

`BucketNotEmpty` appeared once in seven destroys on the versioned bucket, after the Alpine `shared` run. The container's state is thrown away with `--rm`, so the bucket stayed in account 99 until Floci was reset. Cause not verified. Hypothesis: `aws_s3_object` and `aws_s3_bucket_versioning` have no ordering between them, so the object may be written before versioning is on and leave a version or delete marker that the destroy does not remove. Spike 02's `main.tf` adds `depends_on = [aws_s3_bucket_versioning.data]` on the object; I did not run enough destroys to show that it fixes it.
