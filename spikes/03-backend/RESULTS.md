# Spike 03 — S3 backend and locking on Floci

Run 2026-11-13, macOS arm64, Floci 2.4.0 in Docker, Terraform 1.12.2 and 1.17.2, hashicorp/aws 6.73.0. State bucket `tfstate-spike03` (versioning on, `force_destroy = true`) and DynamoDB table `tflock-spike03` (hash key `LockID`) created by `bootstrap/main.tf` in account `000000000031`; the exercise resources (one SSM parameter plus a `terraform_data` that holds the lock) live in the same account, because the backend uses the same credentials as the provider.

## Result: PASS on all four combinations (1.12 / latest × `use_lockfile` / `dynamodb_table`)

### Backend configuration needed

`backend "s3" {}` plus a `-backend-config` file with only `bucket`, `key`, `region` and either `use_lockfile = true` or `dynamodb_table = "tflock-spike03"`. **No** `skip_*`, no `endpoints`, no `use_path_style`: the endpoint comes from `AWS_ENDPOINT_URL=http://localhost:4577` and the credentials from the env. A tflings-generated `-backend-config` file is therefore tiny.

### Behaviour

| Check | 1.12.2 | 1.17.2 |
|---|---|---|
| `init` with `use_lockfile` | OK, no warning | OK, no warning |
| `init`/`apply` with `dynamodb_table` | OK, **no deprecation warning** | OK; `Warning: Deprecated Parameter — The parameter "dynamodb_table" is deprecated. Use parameter "use_lockfile" instead.` |
| `apply`, then `plan -detailed-exitcode` | empty | empty |
| state objects | `spike03/<name>.tfstate` in the bucket | same |
| lock while an `apply` runs | `use_lockfile`: object `<key>.tflock`; DynamoDB: item with `LockID = <bucket>/<key>` (plus a `...-md5` digest item) | same |
| stuck lock (apply killed with `kill -9`) | second run fails with `Error acquiring the state lock` and a `Lock Info:` block (`ID`, `Path`, `Operation`, `Who`, `Version`, `Created`); `terraform force-unlock -force <ID>` → "Terraform state has been successfully unlocked!"; next `plan` empty | same |
| `init -migrate-state -force-copy` local → S3 (both lock modes) | state copied byte for byte (2312 / 2292 bytes), `plan` empty, local `terraform.tfstate` truncated to 0 bytes | same (2347 / 2327 bytes) |

The lock error text differs by mechanism and is a teaching point: `use_lockfile` reports `S3: PutObject ... StatusCode: 412 ... PreconditionFailed`, DynamoDB reports `PutItem ... ConditionalCheckFailedException`. Both then show the same `Lock Info`.

Exam note: `dynamodb_table` is deprecated in 1.17.2 but not flagged by 1.12.2 (the exam version), so the exam era still teaches DynamoDB locking. Both belong in the course.

## Hostnames actually requested (`TF_LOG=DEBUG`, `AWS_ENDPOINT_URL=http://localhost:4577`)

The S3 backend uses **virtual-host style for the state bucket**:

```
http://localhost:4577/                                              (STS, DynamoDB, SSM, IAM ...)
http://tfstate-spike03.localhost:4577/                              (ListObjectsV2)
http://tfstate-spike03.localhost:4577/spike03/<key>.tfstate?x-id=GetObject
http://tfstate-spike03.localhost:4577/spike03/<key>.tfstate.tflock?x-id=PutObject | GetObject | DeleteObject
```

So the backend needs `<bucket>.localhost` to resolve, a second prefixed hostname in addition to spike 01's `<account-id>.localhost` (S3 Control). The AWS SDK switches to path style when the endpoint host is an IP address (`127.0.0.1`), which is why S3 data calls worked on `127.0.0.1` in spike 01 while only the S3 Control call broke (see spike 04).

## Offline (Linux container, `192.0.2.1` as the only nameserver after `init`)

`linux-backend.sh` (Debian 13, Terraform 1.17.2, `use_lockfile`, state created by the native runs):

- online: `init` OK, `plan` empty (so `tfstate-spike03.localhost` resolved through Docker/Lima DNS).
- offline: `getent hosts tfstate-spike03.localhost` → FAIL (and `localhost` itself still resolves from `/etc/hosts`). `plan` fails after **9 s**:
  `Error: Unable to list objects in S3 bucket "tfstate-spike03" ... exceeded maximum number of attempts, 5, ... Get "http://tfstate-spike03.localhost:4577/?list-type=2..."`.
- `AWS_MAX_ATTEMPTS=1 AWS_RETRY_MODE=standard` did **not** reduce this: still "5 attempts". The S3 backend has its own `max_retries` argument (default 5) that wins over the env var (I did not test setting it).

Meaning: with `localhost` as the generic endpoint, the S3 backend breaks offline on Linux, in addition to the provider's S3 Control call.

### With the generic endpoint on `127.0.0.1`: works, including offline

Same checks with `AWS_ENDPOINT_URL=http://127.0.0.1:4577` (no `use_path_style` needed; the SDK uses path style for an IP host):

- native macOS, `init` + `apply` with `TF_LOG=DEBUG`: all 17 requests go to `http://127.0.0.1:4577/`, no `<bucket>.` host.
- Linux container, online and then offline (unreachable nameserver): `plan` → "No changes", 2 s (`linux-backend.sh` with `EP=http://127.0.0.1:4577`).

So the S3 backend does not need DNS at all when the generic endpoint is the IP. See spike 04 for the one call that still does.

## Notes

- Two `migrate.sh` runs failed back to back with `Failed to load plugin schemas ... Failed to read any lines from plugin's stdout` (the AWS provider process did not start); the same runs passed on the next attempt and four more times. Not reproduced, cause unknown; the machine was running several kind clusters at the time.
- A first attempt shared one SSM name and state key across the four runs and failed with `ParameterAlreadyExists`; fixed with a per-run `-var id` and `key=`. Not a Floci issue.
- The stuck-lock test leaves a `sleep 60` child behind after `kill -9`; the script kills it.

## Files

`bootstrap/main.tf`, `app.tf`, `backend-lockfile.hcl`, `backend-ddb.hcl`, `run.sh`, `stuck.sh`, `migrate.sh`, `linux-backend.sh`.
