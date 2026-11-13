# Phase 0 spikes — brief for the agent

Goal: prove or disprove the assumptions in `ROADMAP.md` ("Floci integration" and "Phase 0") before writing any Go. Each spike ends with `RESULTS.md`: what was run (commands), versions, timings, PASS / FAIL / PARTIAL, and what it means for the design. If something fails, propose the alternative; do not silently change the design.

## Setup (once)

1. `spikes/.cache/bin/`: download
   - **Floci 2.4.0** native binary `floci-aws` for this machine from `https://github.com/floci-io/floci/releases` (check the asset names on the release page; verify against `sha256sums.txt`). If the native binary does not run on macOS, fall back to `docker run -p 127.0.0.1:4577:4566 floci/floci:2.4.0` and say so.
   - **Terraform 1.12.x** (latest patch) and **Terraform latest** from `releases.hashicorp.com`, side by side (`terraform-1.12`, `terraform-latest`).
2. `spikes/env.sh`: a script that prints nothing and execs its arguments with a clean env:
   `env -i PATH=... HOME=$PWD/.home AWS_ACCESS_KEY_ID=${ACCOUNT:-000000000001} AWS_SECRET_ACCESS_KEY=test AWS_REGION=us-east-1 AWS_ENDPOINT_URL=http://127.0.0.1:4577 AWS_CONFIG_FILE=/dev/null AWS_SHARED_CREDENTIALS_FILE=/dev/null TF_PLUGIN_CACHE_DIR=$PWD/.cache/plugins "$@"`
3. `spikes/floci.sh up|down|status`: start Floci on 127.0.0.1:4577 with persistent storage in `spikes/.cache/floci-data` (see Floci's `docs/getting-started/installation.md` and `docs/configuration/` for the env vars), wait for `/_floci/health`.

## Spike 01 — AWS provider configured only through the environment

Folder `spikes/01-env-only/`. A config whose provider block is only:

```hcl
provider "aws" {
  region = "us-east-1"
}
```

with `aws` pinned `~> 6.0`, creating: `aws_vpc` + `aws_subnet` + `aws_security_group`, `aws_s3_bucket` + `aws_s3_bucket_versioning` + `aws_s3_object`, `aws_iam_role` + policy attachment, `aws_ssm_parameter`, `aws_secretsmanager_secret` + version, `aws_dynamodb_table`, and a data source (`aws_caller_identity`, `aws_availability_zones`).

Questions to answer, on **both** Terraform versions:
- Does `init` / `plan` / `apply` / `destroy` work with **no** `skip_*` flags, no `endpoints {}` block and no `s3_use_path_style`? If not, which is the minimum that must be added (and can it come from env instead of HCL)?
- Is a second `plan` after `apply` empty (no perpetual diffs)? List any resource that drifts on its own.
- Timings: cold `init` (empty plugin cache), warm `init`, `plan` with no changes (run 5×, give median). Target: warm plan < 5 s.
- Size of the AWS provider in the plugin cache.
- Safety check: run once with the endpoint env var **removed** (keys still fake). Confirm it fails at authentication against real AWS and creates nothing. Do not run this with real keys.

## Spike 02 — one account per exercise

Folder `spikes/02-accounts/`. Same config applied with `ACCOUNT=000000000001` and `ACCOUNT=000000000002`.
- Does `aws_caller_identity` report the two account IDs?
- Can account 2 see account 1's bucket / VPC / SSM parameter? (It should not.) Same bucket name in both accounts — allowed or collision? (S3 names are global in real AWS; record what Floci does.)
- Can one account be wiped without restarting Floci and without touching the other? Look for an admin/reset API in Floci's docs (`/_floci/...`); if none, measure "delete everything via API" with the AWS CLI or a script, or restart-with-snapshot. Record the fastest option.

## Spike 03 — remote state and locking

Folder `spikes/03-backend/`. Create a state bucket (and a DynamoDB lock table with hash key `LockID`) in a dedicated account, then:
- `backend "s3"` with `use_lockfile = true`, endpoint from env only if possible (otherwise the minimal backend args, ideally in a `-backend-config` file the tool would generate).
- Same with `dynamodb_table` (deprecated but still on the exam?) — record any deprecation warning on each Terraform version.
- Leave a lock stuck (kill `terraform apply` mid-run, or hold it with a long `apply` in another shell), show the error a second run prints, then `terraform force-unlock <id>`.
- `terraform init -migrate-state` from local to S3.

## Spike 04 — summary

`spikes/RESULTS.md`: one table (spike, result, decision), then an edit proposal for `ROADMAP.md` ("Facts checked" and "Floci integration") as a diff. Do not apply it to `ROADMAP.md` until the user says so.
