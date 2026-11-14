# CLAUDE.md — tflings

tflings is "rustlings for Terraform": a Go CLI that teaches Terraform (aimed at the Terraform Associate 004 exam) through small broken exercises. Exercises run the real `hashicorp/aws` provider against **Floci**, a local AWS emulator in Docker, so nothing touches a real cloud. It is the sibling of k8slings and golings and copies their shape.

**Status (2026-11-14):** Phase 0 (spikes) is finished. Phase 1 is under way on branch `phase1-skeleton`: the CLI skeleton builds (`init`, `list`, `run`, `hint`, `reset`, `solution`, `update`, `verify`, plain `watch`) and quiz exercises run end to end; plan mode works end to end for exercises that need no Floci: the runner (`internal/engine`, `internal/runner`) and CEL checks (`internal/check`, with `plan`, `config`, `fmt` and `variants` that re-plan with other input to refuse hard-coded answers) run 8 provider-free exercises, proven on Terraform 1.12.2 and 1.17.2. The Floci manager (`tflings aws up|down|status|reset`, `internal/floci`) and `doctor` (`--prefetch` downloads the AWS provider) are built; `resources1` proves the AWS provider path end to end (offline `init` from the cache, plan against Floci, 2.9 s per run). CI workflows are written (`.github/workflows/ci.yml`: lint, test on Linux and macOS, aws with Floci, offline with the network off, windows informational) and the exercise lint test enforces the exercise rules, but all jobs passed on GitHub on 2026-11-14 (Linux, macOS, Windows for the provider-free proofs; Linux with Docker for the AWS-provider proofs). The full-screen `watch` (Bubble Tea v2, adapted from k8slings: hint, solution, reset, edit, next, list, check all; runs report "terraform plan…" while they work) is built and was tried in a real terminal (pty); model tests cover the keys. Apply mode is built (plan checks, apply, `config`/`state` checks, `empty_plan_after`; the state stays in `<home>/runs/<name>/src` between runs and `reset` deletes it; an apply that would hang for want of `<account>.localhost` is stopped with the `/etc/hosts` line). The typed `aws:` checks (`internal/awscheck`) are built: `check.yaml` names one of a fixed list of read-only calls (`s3.versioning`, `ssm.parameter`, `ec2.vpcs` with `is_default` for the default VPC, ...) and asserts on `result` with CEL; adding a call is a few lines of Go and an integration test against Floci. 51 exercises exist (quiz, plan, apply and fix; the `conditions`, `providers`, `versions` and `sensitive` topics are proven on Terraform 1.12.2 and 1.17.2). Variants in `check.yaml` can also `apply_first` (build the configuration with other values, then plan, so a replacement, an ignored change or a forbidden destroy is visible; for provider-free configurations only, linted; the state is deleted afterwards) and `expect_error` (the plan must fail with some text). `aws up` and `watch` empty Floci only while no exercise has a Terraform state with resources. Fix mode is built (see "Fix mode" below): `tflings tf <exercise> <terraform args>` runs Terraform in the exercise's sealed environment, seeding builds the situation once, `run` checks the end state; `refactoring1` (state mv) and `drift1`, `drift2` (refresh, refresh-only) prove it. Not built: `hc-install`, cli mode, the automatic global Floci reset of decision 5 (only the explicit `aws reset` exists). `ROADMAP.md` is the plan, the decisions and the phases. `spikes/RESULTS.md` is the one-page summary of the spikes; each spike's evidence is in `spikes/<nn>-<name>/RESULTS.md`. Facts below say which spike established them. Untested things are marked **untested**.

## Where things are

- `ROADMAP.md` — plan, decisions, phases, risks.
- `spikes/` — Phase 0 experiments and their scripts (`env.sh`, `floci.sh`, per-spike `RESULTS.md`).
- Siblings to copy from: `~/Work/git/k8s/k8slings` (CLI shell, workspace, markers, progress file, CEL checks, CI that proves every exercise) and `~/Work/git/go/golings`.
- Source material for exercises: `~/Work/git/terraform/learn/tf-associate/` (12 days of notes and labs, Apache-2.0, not a git repo).

## Architecture (planned; copy k8slings, replace Kubernetes with Terraform + Floci)

```
cmd/tflings/            entry point (main)
internal/cmd/           cobra commands
internal/engine/        Engine interface (Terraform now, OpenTofu in Phase 8); the only place that execs a terraform binary
internal/runner/        builds the isolated environment, runs init/validate/plan/apply/show with timeouts and output caps
internal/floci/         container lifecycle (docker CLI), health wait, account allocation, reset
internal/check/         check.yaml loader, CEL evaluation over plan/config/state JSON, `aws` SDK read-only calls, lints
internal/exercise/      info.toml manifest, modes, workspace, markers, progress file
internal/tui/           Bubble Tea full-screen watch (built)
exercises/<topic>/<name>/   broken config, README.md, .terraform.lock.hcl; check.yaml, setup/ embedded
solutions/<topic>/<name>/   fixed files or solution.sh, plus shortcut-*.sh the check must refuse
```

Rules for the code: never call `terraform` outside `internal/engine`; keep `engines = ["terraform"]` in every `info.toml` entry until Phase 8; AWS calls for seeding and checks use the AWS SDK for Go v2 pointed at Floci, not `os/exec`.

### Exercise modes

`plan` (init + validate + `plan -out` + `show -json`, then CEL), `apply` (apply, then CEL over state/outputs plus read-only `aws` checks, then a second plan must be empty), `fix` (seeded broken situation, end state checked), `cli` (one command in `answer.sh`, allowlist parser), `quiz` (`quiz.toml`, options shuffled). All but `quiz` need Terraform and Docker. `info.toml` carries `objective = "4e"` (exam content-list id) and `engines`.

## Fix mode

A fix exercise is a situation that has already gone wrong, repaired with Terraform commands.

```
exercises/<topic>/<name>/   README.md (the ticket), main.tf (the configuration as it is now; the learner may edit it),
                            check.yaml and setup/ (embedded only; init never writes setup/, not even empty folders)
  setup/steps.yaml          steps: `apply: <folder>` (files in setup/<folder>, then init + apply -auto-approve) or
                            `mutate: {call, args}` (an allowlisted change in Floci behind Terraform's back, `internal/awscheck/seed.go`)
solutions/<topic>/<name>/   solution.sh (one `terraform ...` line per command, parsed by tflings, never run by a shell), plus any .tf the solution changes
```

- `tflings tf <name> <args>` builds the situation on first use (marker file `seeded` in the run folder), copies the learner's current files over the setup's, runs `init`, then runs the real Terraform with the learner's terminal attached (`runner.Attach`, no timeout) in the sealed environment. `tflings env <name>` prints that environment.
- The state lives in `<home>/runs/<name>/src` and survives between commands; `reset` forgets it (and bumps the account counter), so the next command seeds again in a fresh account.
- `run` never applies anything. It plans with refresh (`-detailed-exitcode`), reads the state, evaluates `plan`/`config`/`state`/`aws` checks, and with `empty_plan_after: true` requires nothing left to change. `seed` in a `state` expression is the state right after seeding, so a check can tell a renamed object from one destroyed and built again (`refactoring1`).
- A fix exercise needs Floci only when its `info.toml` says `requires = ["floci"]`; a `mutate` step without it is refused by the lint. The proof (`RunSolution`) seeds, requires the check to fail, plays `solution.sh`, and requires the check to pass.
- `check.yaml` can also have `files:` assertions over `files` (file name to text) for what a plan cannot show, such as a `moved` block (`refactoring2`) or the operator of a version constraint (`versions1`). Seeding calls so far: `ssm.put_parameter`, `ssm.delete_parameter`, `s3.create_bucket`, `s3.put_object` (the last two make a bucket by hand, `import1`, and write the lock file of a dead run, `locking1`). A fix exercise built in Floci counts as saved state (the `seeded` marker says `floci`), so `aws up` and `watch` do not empty Floci under it; `reset` clears it.
- `make ci` runs vet and the whole suite once per Terraform in `TF_BINS` with every exercise proven (Floci and a provider cache needed for the AWS ones); it is what to run while GitHub Actions is off.
- The SSM provider hides `value` as `(sensitive value)` in a plan, so a learner of `drift1` sees that the parameter changes but not to what.

## Runner environment (the safety boundary)

**Never touch real AWS.** Every Terraform or AWS CLI command runs from a from-scratch environment (`env -i`): only `PATH`, `HOME`, the exercise's fake keys, the two endpoint variables, `AWS_CONFIG_FILE=/dev/null`, `AWS_SHARED_CREDENTIALS_FILE=/dev/null`, `TF_PLUGIN_CACHE_DIR`/`TF_CLI_CONFIG_FILE` and tflings's own `TF_DATA_DIR`. Every other `AWS_*`, `TF_*`, `ARM_*`, `GOOGLE_*` is dropped. Never use the user's AWS profiles, SSO sessions or `~/.aws`. If a command would need real credentials, stop and ask.

```
AWS_ACCESS_KEY_ID=<12-digit account>     # one account per exercise, see below
AWS_SECRET_ACCESS_KEY=test
AWS_REGION=us-east-1
AWS_ENDPOINT_URL=http://127.0.0.1:4577
AWS_ENDPOINT_URL_S3_CONTROL=http://localhost:4577
```

- **Why split endpoints** (spikes 01, 03, 04): the AWS provider's S3 Control call (bucket tags) prepends `<account-id>.` to the endpoint host even when `AWS_ENDPOINT_URL_S3_CONTROL` is set, and `<account>.127.0.0.1` cannot resolve. `localhost` makes it `<account>.localhost`. With the generic endpoint on the IP, S3 data calls and the S3 state backend use path style and need no DNS. The provider block in exercises is only `region`; no `skip_*`, no `endpoints {}`, no `use_path_style`.
- **Online-first** (decision 10): `<account>.localhost` must resolve. It does where the resolver answers `*.localhost` (Docker/Lima DNS did; macOS and systemd-resolved are **untested**). With no DNS, `apply` hangs. `doctor` resolves it and, on failure, prints the exact `sudo` `/etc/hosts` lines (`127.0.0.1 <account>.localhost`); the lines fixed it in spike 04. No root-free fix is planned.
- **Fast failure** (spike 04): `AWS_MAX_ATTEMPTS=1` did **not** stop the offline hang, and the S3 backend still made 5 attempts. The runner must put its own wall-clock timeout around every Terraform call and print a clear message.
- Before every `plan`/`apply`, a static check of the learner's `provider "aws"` blocks refuses `profile`, `assume_role`, `shared_*_files`, real-looking keys and `endpoints` that do not point at the tflings Floci. Even without it, fake keys with the endpoint removed fail at STS `InvalidClientTokenId` and write no state (spike 01; that check sends one STS call with a fake key to real AWS).

## Floci

- Run in **Docker only** (Floci 2.4.0 published no native binary), image **pinned by digest** `floci/floci@sha256:5ee85db604c4aa74ac76e07ea5cc83086f37bb472bf3dbb97e34019e48efae9c`, container `tflings-floci`, published on `127.0.0.1:4577` (not 4566, which the user may use), data in the named volume `tflings-floci-data`. A bind mount does not work under Lima (`$HOME` is read-only to containers).
- Recipe (spike 05; the hardened flags are verified, not yet in `spikes/floci.sh`):
  ```
  docker run -d --name tflings-floci -p 127.0.0.1:4577:4566 \
    -e FLOCI_STORAGE_MODE=persistent -e FLOCI_STORAGE_PERSISTENT_PATH=/app/data \
    -v tflings-floci-data:/app/data \
    --cap-drop ALL --cap-add SYS_CHROOT --cap-add SETUID --cap-add SETGID \
    --security-opt no-new-privileges --read-only --tmpfs /tmp \
    floci/floci@sha256:5ee85db6...   # full digest above
  ```
  The volume must be at `/app/data`: the image declares a volume there and any other mount leaks an anonymous volume per run. Never mount the Docker socket. Wait with the container's health status or `/_floci/health` (0.07 s to healthy, 29 MiB idle). `tflings aws up` inspects the existing container first (reuse / start / recreate on a different digest); do not rely on Docker's error text for port or name conflicts.
- Only use services proven by our own CI or in Floci's Terraform compatibility suite: S3, VPC/subnet/security group, IAM, SSM, Secrets Manager, DynamoDB, Route 53, KMS. Container-backed services (Lambda, ECS, EC2 instances) are out of scope.

## Accounts and reset (spike 02, decision 5)

- One account per exercise: a 12-digit access key ID is the account ID, resources are invisible across accounts, and the same names are allowed in two accounts.
- The account ID is deterministic from the exercise name plus a **reset counter kept in the progress file**. `reset` bumps the counter and gets a fresh empty account. There is no single-account reset API: the scripted per-account wipe (8.8 s) is a fallback only, and the global `POST /_floci/state/reset` runs only on `tflings aws up` or when `watch` starts with no active exercise, after a warning.
- Every account starts with a default VPC; the check DSL gets a `non_default` helper.

## Terraform, providers, cache

- Versions: floor Terraform 1.12 (the exam version), CI on 1.12 and latest; AWS provider `~> 6.0` (6.73.0 in the spikes). Pin Terraform and provider versions in writing; CI proves every exercise ("broken fails, solution passes").
- The AWS provider is 729 MB. `doctor --prefetch` fills a shared cache by running `init` once online. **The plugin cache alone does not make `init` work offline** (Terraform still asks the registry). The runner writes a CLI config (`TF_CLI_CONFIG_FILE`) with `filesystem_mirror { path = "<cache>" include = ["hashicorp/aws"] }` plus `direct { exclude = ["hashicorp/aws"] }`; **the mirror folder must not also be `TF_PLUGIN_CACHE_DIR`** (Terraform fails with "cannot install existing provider directory ... to itself"), so runs use `<home>/plugins` as the mirror and `<home>/plugins-extra` as the cache for other providers. Then `init` is offline and 0.4 s, and each exercise's `.terraform` is about 30 KB (symlink). Ship exercise lock files with `zh:` hashes only (`resources1` does; embed uses `all:` so dotfiles like `.terraform.lock.hcl` are included). Spike 07 verified the mirror on Linux; `resources1` verified it on macOS; Windows is **untested**.
- Plan-per-save budget (spike 07, macOS): `validate` + `plan -out` + `show -json` takes 3.1 s (no resources), 3.8 s (1), 5.1 s (13); each command costs about 1 s for the provider start. Plan-mode checks should use `-refresh=false` (about 1.3 s saved at 13 resources). Plan mode runs `init` then `plan -out` then `show -json` and no separate `validate`: init and plan report every syntax and type error the broken exercises have, and the `configuration` part of the plan JSON carries variables, outputs and resources for the `config` checks (verified by the 8 plan exercises on Terraform 1.12.2 and 1.17.2).
- Remote state (spike 03): `backend "s3" {}` plus a generated `-backend-config` file with only `bucket`, `key`, `region` and `use_lockfile = true` or `dynamodb_table`. Stuck lock, `force-unlock` and `init -migrate-state` all work on 1.12.2 and 1.17.2. `dynamodb_table` warns as deprecated on 1.17.2 but not on 1.12.2 (the exam version), so both lock modes are taught.

## Exercise rules (checked by a lint in the exercise checker where noted)

- `force_destroy = true` on every `aws_s3_bucket` (lint).
- `recovery_window_in_days = 0` on every `aws_secretsmanager_secret`, or a destroyed secret blocks recreating the name.
- No "bucket name already taken" exercises: Floci's S3 namespace is per account. Those are quiz questions, and the README says so.
- Prefer ordering the S3 object after the versioning resource (`depends_on`); a destroy once failed with `BucketNotEmpty`, cause not verified (spike 00).
- `# I AM NOT DONE` on line 1 until fixed, `# TODO:` at the fix site, lesson in `README.md` with a link to the matching HashiCorp tutorial, `objective` set.
- Assume an exercise's resources may be left behind by an interrupted apply; checks must not depend on a clean state.

## Floci differences from AWS found so far

Each is a thing an exercise must avoid or a check must tolerate; report upstream when time allows.
- Floci 2.4.0 accepts S3 bucket names that AWS rejects (an uppercase name applied without error), so "invalid bucket name" cannot be taught by an apply failure.
- **S3 bucket tags:** after Terraform changed a bucket's tags from one tag to three, Floci returned only the two tags that were added (the first was gone), and Terraform's own state agreed with Floci, not with the configuration. AWS merges. Observed once on Floci 2.4.0; the cause was not investigated. Exercises that add tags use `aws_vpc`, where adding tags in two steps worked.
- S3 buckets are isolated per account (spike 02), unlike AWS.
- **Secrets Manager ARNs resolve across accounts** (observed 2026-11-14 on Floci 2.4.0): a refresh in account B of a state written in account A found A's secret. Only a test harness that changes the account salt but keeps the state can meet this; the proof test now forgets the saved run before and after each exercise.
- **Write-only arguments work** (2026-11-14, Floci 2.4.0, AWS provider 6.73.0, Terraform 1.12.2 and 1.17.2): an ephemeral variable into `aws_ssm_parameter.value_wo` and `aws_secretsmanager_secret_version.secret_string_wo` applies, the value is not in the state, and the next plan is empty (`sensitive3`, `sensitive4`).

## Exam facts (spike 06)

Terraform Associate 004 tests Terraform 1.12; 1 hour, multiple choice, online proctored, $70.50, credential valid 2 years. HashiCorp does not publish the question count or pass mark, so `tflings exam` makes the count a setting and labels any default unofficial. HCP Terraform Free: 500 managed resources, unlimited users, one concurrent run (IBM support page, 2026-06-23; not on HashiCorp's pricing page). Remote HCP runs cannot reach a local Floci, so the optional HCP track needs local execution mode or resources that do not need Floci (**untested**); objective 8 is covered by quiz anyway.

## Linux results (2026-11-14, run by hand while GitHub Actions was out of minutes)

Every exercise proof passed, AWS-provider exercises against Floci included, on:
- Linux arm64, Ubuntu 26.04 in a Lima `vz` VM with native Docker (Floci 2.4.0), Terraform 1.12.2 and 1.17.2: 375 s cold, 131 s warm (2 proofs at a time). `*.localhost` resolved to `::1` with no `/etc/hosts` line.
- Linux x86_64, Debian 13 on an i9-9980HK with native Docker, Terraform 1.12.2 (1048 s) and 1.17.2 (1053 s). The provider takes about 5 s to start there, against about 1 s on an M-series Mac, so one AWS exercise takes 36 to 90 s and Go's default 10 minute test timeout is too short: use `-timeout 60m`.
- To test on Linux without a Go toolchain there (the Lima VM ran out of memory compiling the AWS SDK, with kind clusters alongside): cross-compile on the Mac with `GOOS=linux GOARCH=arm64 go build ./cmd/tflings` and `go test -c ./internal/exercises` (the exercises are embedded in the test binary), copy both, run `tflings doctor --prefetch` there, then run the test binary with `TFLINGS_TEST_HOME` and `TFLINGS_TEST_TERRAFORM` set.
- The proofs run in parallel (`TFLINGS_TEST_PARALLEL`, default half the cores): on macOS 170 s one at a time, 54 s with 4, 42 s with 8.

## Not yet tested

Windows with Docker and the AWS-provider exercises, macOS with the network off, systemd-resolved, Docker Desktop/OrbStack for the Floci container (Lima and Linux Docker are covered above), the offline mirror recipe on Windows. The GitHub Actions jobs (macOS, Windows, offline) last ran on 2026-11-14 for the `sensitive` PR; Actions is disabled until 2026-12-01 (private repo, 2000 free minutes used), and `make ci` stands in for it on macOS.

## Conventions

- Facts you checked go into a `RESULTS.md` with the date and exact versions. Do not write anything as fact that you did not run.
- Downloads go under `spikes/.cache/` (git-ignored); verify checksums when the project publishes them.
- Conventional commit messages, one branch per piece of work (`phase0-spikes` for Phase 0), no push unless asked, no attribution lines in commit messages.
- Go floor 1.26, Apache-2.0. Floci is MIT and is downloaded, not vendored; credit it in `NOTICE` and the README.
