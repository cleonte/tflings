# Spike 02 — one account per exercise

Run 2026-11-13, macOS arm64, Floci 2.4.0 in Docker, Terraform 1.17.2, hashicorp/aws 6.73.0, AWS CLI 2.40.2 (`amazon/aws-cli` container, same clean env, host network inside the Lima VM). Config: spike 01's with a **fixed** bucket name (`spike02-same-name`) so both accounts use identical names for the bucket, SSM parameter, IAM role/policy, secret and DynamoDB table.

## Result: PASS for isolation; no single-account reset API

### Account IDs

`ACCESS_KEY=000000000001` and `000000000002` (12 digits, secret `test`). `aws_caller_identity` outputs `000000000001` and `000000000002`; every ARN carries its own account ID (e.g. `arn:aws:ssm:us-east-1:000000000002:parameter/spike01/greeting`). Terraform 1.17.2 `plan` is empty in both after apply. Unmodified `main.tf` apart from the names; no HCL needs to know the account.

### Isolation

- Account 2 cannot see account 1's resources: `describe-vpcs --vpc-ids <acct1 vpc>` → `InvalidVpcID.NotFound`; `head-object` on an object that exists only in account 1 → 404; `get-parameter /only-in-1` → `ParameterNotFound`. Bucket listing, objects, IAM roles, secrets, DynamoDB tables and SSM parameters each show only that account's own.
- **Same bucket name in both accounts: allowed**, no collision. Floci's default S3 namespace is per account (unlike real AWS, where bucket names are global). Floci has `FLOCI_SERVICES_S3_GLOBAL_BUCKET_NAMESPACE=true` for the global behaviour (from its docs; not tried). Consequence: an exercise about "bucket name already taken" cannot be taught across accounts by default.
- Every account also has a default VPC (`vpc-default-us-east-1`) before anything is created; `describe-vpcs` shows 1 in a fresh account, so counts in checks must exclude it.
- A fresh account ID costs nothing: no setup, nothing to delete.

### Wiping one account

| Option | Time | Scope | Verdict |
|---|---|---|---|
| Floci admin API `POST /_floci/state/reset` | 0.05 s | **all accounts** (both accounts empty afterwards) | no account parameter exists (`EmulatorInfoController.reset()` takes none; the docs list a `/state/nuke` too; not tried). Documented as refused with 409 under the DynamoDB-Local backend; this image's default backend accepted it. |
| Scripted delete through the AWS API (`wipe-account.sh`, one CLI container) | 8.8 s for the 13 resources | one account; the other account was untouched (buckets/SSM/DDB/roles/secrets/VPC counts unchanged) | works; needs a per-service deletion list that grows with the exercises; most of the time is container start and CLI calls, a Go implementation would be faster |
| `terraform destroy` with the state | 3.4 s (spike 01) | one account | only when the state still exists |
| Restart container with persistent volume | 0.45 s to healthy | all accounts, and **state survives** (both accounts intact after) | useful for a "Floci crashed" check, not a wipe |
| Per-account snapshot/restore | not possible | — | persisted files are per service (`ec2-security-groups.json`, `dynamodb-tables.json`, ...), with accounts as key prefixes inside, so one account cannot be removed by deleting files |

After either wipe the stale Terraform state refreshes cleanly: `plan` shows `13 to add`, no errors.

### Design implications

1. **Use a distinct account ID per exercise** (e.g. derived from the exercise number). No clean-up needed between exercises, learners can leave and come back (persistent volume).
2. `tflings reset <exercise>`: either use a fresh account ID for the exercise (instant, but leaves orphans), or run the API wipe for that account (8.8 s with a CLI; a Go implementation using the SDK should be quicker, not measured). Prefer the fresh ID plus an occasional global `/_floci/state/reset` when no exercise is active (e.g. on `tflings init`, with confirmation).
3. A global reset is one 50 ms call, so CI can reset between exercises. Do not run it while a learner has an exercise in progress.
4. Because of the per-account S3 namespace, bucket-name collisions are not teachable; use names containing the account ID (spike 01) or per-exercise names.

## Files

`main.tf`, `run.sh <account> <terraform args>` (own work dir/state per account), `awscli.sh <account> <aws args>`, `wipe-account.sh <account>`.
