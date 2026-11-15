# ROADMAP — tflings

tflings is a Rustlings-style course for Terraform, built the way [k8slings](https://github.com/cleonte/k8slings) and [golings](https://github.com/cleonte/golings) are built: same CLI shape, `info.toml` manifest, `exercises/` + `solutions/` trees, `# I AM NOT DONE` marker, Bubble Tea `watch`, `doctor`, CI that proves every exercise ("broken fails, solution passes"). This file is the plan; `CLAUDE.md` (written in Phase 0) will be the architecture.

The syllabus follows the **HashiCorp Terraform Associate (004)** study guide and exam content list. The exercises use the **real `hashicorp/aws` provider against [Floci](https://floci.io/aws/)**, a free, local AWS emulator, so learners write the same AWS Terraform they would write at work, with no AWS account, no cost and no risk to a real account.

## Facts checked while writing this plan (2026-11-13)

- Terraform Associate **004** tests **Terraform 1.12** and includes HCP Terraform content. ([study guide](https://developer.hashicorp.com/terraform/tutorials/certification-004/associate-study-004), [exam content list](https://developer.hashicorp.com/terraform/tutorials/certification-004/associate-review-004))
- Current Terraform is 1.16.x (1.17.0 just out). **Floor 1.12 (the exam version), CI on 1.12 and latest.**
- LocalStack's free Community edition was sunset in March 2026 (account required now), so it is not an option.
- **Floci** (`github.com/floci-io/floci`, formerly `hectorvent/floci`):
  - **MIT**, no paid tier. Latest release **2.4.0** (2026-10-31); very active (commits daily), so **pin a version** and watch releases.
  - Ships as a Docker image **and a native binary** `floci-aws` (~40 MB, macOS/Linux/Windows), listens on `127.0.0.1:4566`, ~24 ms start, ~13 MiB idle, persistent storage optional, snapshots.
  - Without Docker, in-process services work fully (S3, IAM, STS, VPC/subnets/security groups/route tables, SSM, Secrets Manager, KMS, DynamoDB, SQS, SNS, Route 53, CloudWatch Logs, ACM...). Container-backed ones (EC2 instances, RDS, EKS...) keep their control plane but start nothing; Lambda/ECS fail. EC2 `RunInstances` with Docker launches a real container.
  - Its own CI runs a **Terraform compatibility suite** (Terraform 1.14.7, AWS provider `~> 6.0`, ~60 resource types incl. VPC, IAM, S3, DynamoDB, SSM, Secrets Manager, Route 53, KMS) and an **OpenTofu** one, and tests the **S3 backend with DynamoDB locking**.
  - **Multi-account isolation**: a 12-digit access key ID is used as the account ID, and each account's resources are invisible to the others. That gives each exercise its own clean "AWS account".
  - A web console at `/_floci/ui` (needs Docker).
- **Spike 01 (2026-11-13, macOS arm64, Floci 2.4.0 in Docker, Terraform 1.12.2 and 1.17.2, `hashicorp/aws` 6.73.0) — PASS** (`spikes/01-env-only/RESULTS.md`):
  - `init`/`plan`/`apply`/`destroy` work with a provider block of only `region`, everything else from env; no `skip_*`, no `endpoints {}`. Second plan empty, no perpetual diffs.
  - S3 Control prefixes the account ID to the endpoint host, so a bare `127.0.0.1` endpoint breaks bucket tag reads (`000000000001.localhost` resolved on macOS). Superseded by spike 04: split endpoints, IP for the rest and `localhost` for S3 Control.
  - Warm plan ~2.9 s (13 resources), warm init ~1.25 s, cold init ~11 s. AWS provider = **737 MB** per version/platform in the plugin cache.
  - Endpoint removed + fake keys → fails at STS `InvalidClientTokenId`, no state written (one STS call with a fake key reaches real AWS).
  - Secrets Manager soft-deletes: every exercise secret needs `recovery_window_in_days = 0`.
  - **Floci 2.4.0 published no native binary** (release has no assets), so Docker was needed. Under Lima `$HOME` is read-only to containers, so Floci data lives in a named Docker volume.
- To verify in Phase 0: exam format (question count, duration, types), HCP Terraform free tier, AWS provider env-only configuration against Floci (see spikes).

## The eight objectives → topics

| Obj | Exam objective | tflings topics |
|---|---|---|
| 1 | IaC concepts (1a–1c) | `iac` (quiz) |
| 2 | Fundamentals: providers, versions, multiple providers, state (2a–2d) | `providers`, `versions`, `state-basics` |
| 3 | Core workflow: init, validate, plan, apply, destroy, fmt (3a–3g) | `workflow`, `fmt` |
| 4 | Configuration (4a–4h) | `resources`, `data-sources`, `references`, `variables`, `outputs`, `types`, `expressions`, `functions`, `iteration`, `dependencies`, `lifecycle`, `conditions`, `sensitive` (sensitive, ephemeral, write-only, Secrets Manager / SSM) |
| 5 | Modules (5a–5d) | `modules`, `module-versions` |
| 6 | State management (6a–6d) | `backends` (S3 backend on Floci), `locking` (`use_lockfile`, DynamoDB), `drift`, `refactoring` |
| 7 | Maintain infrastructure (7a–7c) | `import`, `inspect`, `debugging` |
| 8 | HCP Terraform (8a–8d) | `hcp` (quiz; hands-on optional, needs an HCP account) |
| — | Capstone | `capstone`: a broken VPC + S3 + IAM stack across modules, with drift and a stuck lock |

Planning size: **~120 exercises in ~25 topics**, plus a question bank for the mock exam.

## Design

### Exercise modes

| Mode | What the learner does | How it is checked |
|---|---|---|
| **plan** | Fix `main.tf` (and friends) | `init` + `validate` + `plan -out` → `terraform show -json` → CEL assertions over plan and configuration |
| **apply** | Fix the config so it really builds | Apply against Floci, then CEL over state, outputs and **`aws` checks** (what really exists in Floci, read with the AWS SDK) |
| **fix** | Repair a seeded broken situation with the CLI (drift, stuck lock, resource renamed in code, bucket created by hand that must be imported) | End state in Floci + Terraform state; a second plan must be empty |
| **cli** | Write one command in `answer.sh` (`terraform state show ...`, a `console` expression, `output -raw ...`) | Allowlist parser (like k8slings query mode), compare with the reference answer, variant against hard-coding |
| **quiz** | Edit `quiz.toml`, set `answer = "b"` (or several) | Answer key; explanation shown after a pass; options shuffled |

Every mode except `quiz` needs `terraform` (fetched and pinned by tflings if missing) and Docker (for Floci). Without Docker, `quiz` exercises still run and the rest are skipped with the fix `doctor` prints, like k8slings's cluster exercises without kind.

### Floci integration (replaces the "fake cloud")

- **tflings runs its own Floci in Docker**, never the learner's: `tflings aws up|down|status` runs the Floci image **pinned by digest** as container `tflings-floci`, published on `127.0.0.1:4577` (a tflings-only port), data in the named volume `tflings-floci-data` (works under Lima/Colima, where `$HOME` is read-only to containers). The endpoints are split (spikes 01, 03, 04): `AWS_ENDPOINT_URL=http://127.0.0.1:4577` (S3 data calls and the S3 state backend then use path style and need no DNS) and `AWS_ENDPOINT_URL_S3_CONTROL=http://localhost:4577` (the AWS provider's bucket-tag call prepends `<account-id>.` to the host, so `<account-id>.localhost` must resolve). `run`/`watch` start it on demand. Docker is required, as it is for k8slings (Docker Desktop, OrbStack, Colima, Lima or Docker on Linux); the native `floci-aws` binary stays a possible later option, not a plan.
- **One AWS account per exercise**: each exercise gets a fixed 12-digit access key ID (`AWS_ACCESS_KEY_ID=<account>`), so exercises never see each other's resources. The account ID is deterministic from the exercise name plus a reset counter kept in the progress file: `reset` bumps the counter and gets a fresh, empty account (spike 02). The scripted per-account wipe (8.8 s with the AWS CLI) is a fallback only.
- **Never real AWS**, enforced in three layers:
  1. The runner builds Terraform's environment from scratch: `AWS_ENDPOINT_URL=http://127.0.0.1:4577` and `AWS_ENDPOINT_URL_S3_CONTROL=http://localhost:4577` (see spike 04), the exercise's fake keys, `AWS_CONFIG_FILE` and `AWS_SHARED_CREDENTIALS_FILE` pointing at tflings's own empty files, every other `AWS_*`, `TF_*`, `ARM_*`, `GOOGLE_*` dropped. A real profile or SSO session is never read.
  2. Before every `plan`/`apply`, a static check of the learner's `provider "aws"` blocks refuses `profile`, `assume_role`, `shared_*_files`, real-looking keys, and `endpoints` that do not point at the tflings Floci.
  3. Even if both failed, the fake keys would be rejected by real AWS.
- **Provider download**: the AWS provider is large (hundreds of MB unpacked). tflings sets one shared `TF_PLUGIN_CACHE_DIR`, ships `.terraform.lock.hcl` files pinned to one AWS provider version, and `doctor --prefetch` downloads it once; after that the course works offline. Version-constraint and lock-file lessons use small providers (`hashicorp/random`, `hashicorp/local`) with several versions, so they stay fast.
- **Commands for playing with it**:
  - `tflings aws <args>` — the AWS CLI against the exercise's account in Floci (like `k8slings kubectl`); if the AWS CLI is not installed, a small built-in `tflings aws ls|show|tag|delete` covers what the exercises need.
  - `tflings drift <exercise>` — change things behind Terraform's back ("someone clicked in the console").
  - `tflings console` — opens Floci's web console (Docker).
  - `tflings sandbox` — a scratch folder wired to Floci (own account) for free play outside the exercises.
- **Seeding and checks use the AWS SDK for Go v2** pointed at Floci (no `os/exec` for AWS calls), as k8slings uses client calls rather than shelling out where it can.

### Checks (`check.yaml`)

Same CEL approach as k8slings, over Terraform's JSON and over what Floci reports:

```yaml
plan:
  - expr: plan.resource_changes.exists(r, r.address == "aws_s3_bucket.logs" && "create" in r.change.actions)
    message: "The plan should create aws_s3_bucket.logs."
config:
  - expr: config.root_module.variables["region"].default == "eu-west-1"
state:
  - expr: state.values.outputs["vpc_id"].value.startsWith("vpc-")
aws:
  - call: s3.GetBucketVersioning { Bucket: "tflings-logs" }
    expr: result.Status == "Enabled"
    message: "Versioning should be enabled on the logs bucket."
empty_plan_after: true          # apply/fix: a second plan must have no changes
```

`aws` calls come from an allowlist of read-only SDK calls. `check.yaml`, `setup/` and `solutions/` are embedded and never written to the workspace.

### Exercise layout

```
exercises/<topic>/<name>/
  README.md        lesson (few lines), links to the matching HashiCorp tutorial
  main.tf ...      the broken configuration; # TODO: at the fix site; # I AM NOT DONE on line 1
  .terraform.lock.hcl   pinned providers (so init is reproducible and offline after prefetch)
  quiz.toml        quiz mode only
  answer.sh        cli mode only
  check.yaml       embedded only
  setup/           fix mode only: what to seed in Floci, a tfstate, a lock (embedded only)
solutions/<topic>/<name>/   fixed files, or solution.sh for fix mode, plus shortcut-*.sh the check must refuse
```

`info.toml` entries carry `objective = "4e"`, `requires = [...]` (e.g. `hcp`) and `engines = ["terraform"]`.

### Commands

Same as k8slings (`init`, `watch`, `run`, `hint`, `list`, `verify`, `reset`, `solution`, `update`, `doctor`), with `cluster` replaced by `aws up|down|status`, plus `aws`, `drift`, `console`, `sandbox`, `terraform ...` (pass-through with the isolated env), `exam` (timed mock exam, scored per objective) and `progress` (readiness per objective).

## Decisions (2026-11-13)

1. **Floci, against AWS.** Exercises use the real `hashicorp/aws` provider and AWS resources on Floci. No own fake cloud, no real AWS/Azure track. (The optional HCP track is the only part that needs an account; it is skippable and comes last.)
2. **Terraform first, OpenTofu-ready.** The runner talks to an `Engine` interface from day one; `tofu` support is Phase 8.
3. **Repo `github.com/cleonte/tflings`, Apache-2.0, Go floor 1.26**, like golings and k8slings. Floci is MIT and is downloaded, not vendored; credit it in `NOTICE` and the README.
4. **Source material: `~/Work/git/terraform/learn/tf-associate`** (see below).
5. **Account per reset** (spike 02): a fresh account ID per reset, deterministic from exercise name + reset counter in the progress file. The 8.8 s scripted wipe is a fallback only. The global `POST /_floci/state/reset` runs on `tflings aws up`, or when `watch` starts with no active exercise, after a warning.
6. **No "bucket name already taken" exercises**: Floci's S3 namespace is per account, so the situation cannot happen. Those become quiz questions, and the README says so.
7. **Default VPC**: every account starts with one, so the check DSL gets a `non_default` helper.
8. **`force_destroy = true` on every bucket** in every exercise template, enforced by a lint in the exercise checker.
9. **Retries**: the runner sets `AWS_MAX_ATTEMPTS` low, `AWS_RETRY_MODE=standard`, a short timeout and a clear error message, so a learner never waits minutes on retries. Spike 04 showed `AWS_MAX_ATTEMPTS=1` does **not** bound the offline failure (apply still running after 90 s; the S3 backend still made 5 attempts), so the runner also needs its own wall-clock timeout around every Terraform call.
10. **Online-first** (2026-11-14): learners need normal network access and DNS. If `<account>.localhost` does not resolve (minimal Linux, restricted resolver, offline), `doctor` detects it and prints the exact `sudo` `/etc/hosts` lines. No root-free offline fix is planned; the plugin cache can still be prefilled with `doctor --prefetch`.

## Source material: `learn/tf-associate`

A 12-day plan (day 0 to day 12) already mapped to the 004 objectives. It is Cristian's own; it has no LICENSE file, so add one there (or state in `NOTICE` that it is the author's own) before adapting. Its labs use `local`/`random`/`terraform_data`; they are rewritten for AWS on Floci (S3, VPC, IAM, SSM, Secrets Manager) so they can be checked and broken.

| Source | Becomes |
|---|---|
| `day-00-hcl-basics` | `hcl` intro topic, plan mode |
| `day-01-workflow` | `iac` quiz + `workflow`, `fmt` |
| `day-02-providers` | `providers` (aws + aliases for two regions), `versions` |
| `day-03-state` | `state-basics`, first `drift` exercises |
| `day-04-resources-dependencies` | `resources`, `data-sources`, `references`, `dependencies`, `lifecycle` |
| `day-05-variables` | `variables`, `outputs`, `types` |
| `day-06-expressions` | `expressions`, `functions`, `iteration` |
| `day-07-conditions-secrets` | `conditions`, `sensitive` (ephemeral `aws_secretsmanager_secret_version`, write-only args) |
| `day-08-modules` | `modules`, `module-versions` |
| `day-09-backends-state-locking` | `backends`, `locking` (S3 backend + `use_lockfile` / DynamoDB, all on Floci) |
| `day-10-maintain` | `import`, `refactoring`, `inspect`, `debugging` |
| `day-11-hcp-terraform` | `hcp` quiz (+ optional hands-on track) |
| `day-12-mock-exam`, `practice-questions.md` | seed of the question bank (30 questions; their key is almost all "b", so options are shuffled) |
| `extras/cheatsheet`, `glossary`, `azure-aws-real-world` | hint texts, docs links, "at work" notes in READMEs |

## OpenTofu (Phase 8)

Easier with Floci than with a custom provider: Floci already tests OpenTofu in its CI, and the AWS provider and S3 backend work in both. Remaining cost (~10%): the `Engine` abstraction (binary, version, install), tagging exercises whose feature differs (HCP / `cloud` block is Terraform-only; state encryption, provider `for_each`, `-exclude` are OpenTofu-only), and a CI matrix of Terraform 1.12, Terraform latest, OpenTofu latest.

## Phases

### Phase 0 — Decisions and spikes
- [x] Floci against AWS; Terraform first behind an `Engine` interface; repo, license, Go floor.
- [x] Spike 01: AWS provider configured only through env — PASS (endpoint must be `localhost`).
- [x] Decided: Floci runs in Docker (pinned by digest, named volume); Docker is required, as for k8slings.
- [~] Spike 00 (partial): `<account>.localhost` resolves on Linux (Debian glibc, Alpine musl) with DNS available; **fails with no DNS** (offline). Windows, macOS offline, amd64, systemd-resolved untested.
- [x] Spike 02: one account per exercise via 12-digit key ID — PASS; no single-account reset API (global `/_floci/state/reset` only), so `reset` uses a fresh account ID.
- [x] Spike 04: `AWS_ENDPOINT_URL_S3_CONTROL` is honoured but the account-ID prefix stays; splitting the endpoints (IP for the rest, `localhost` for S3 Control) leaves one name per account to resolve.
- [x] Spike 03: S3 backend with `use_lockfile` and with DynamoDB, stuck lock + `force-unlock`, `init -migrate-state` — PASS on 1.12.2 and 1.17.2.
- [ ] CI job with the network disabled during `apply` (expected to need the `/etc/hosts` lines; proves `doctor`'s message).
- [ ] CI matrix for Windows, macOS offline, Linux amd64 and systemd-resolved.
- [~] Spike 05 (partial): tflings-managed Floci container on macOS + Lima — PASS (0.07 s to healthy, 29 MiB idle, state survives container removal, hardened run works, named volume must be at `/app/data` to avoid anonymous-volume leaks). Native Linux, Docker Desktop, Colima, OrbStack, Windows untested.
- [x] Spike 07: plugin cache alone does NOT give offline `init` (Terraform still asks the registry); `filesystem_mirror`/`-plugin-dir` pointed at the cache does (0.4 s, 28 KB `.terraform`, zh-only lock files are enough). Plan per save: 3.1 s (no resources) to 5.1 s (13 resources) with validate + plan + show; `-refresh=false` saves about 1.3 s on 13 resources.
- [x] Spike 06: exam is 1 h, multiple choice, online proctored, $70.50, Terraform 1.12, credential valid 2 years; question count and pass mark are not published by HashiCorp. HCP Free: 500 managed resources, unlimited users, 1 concurrent run (IBM support page, 2026-06-23; not shown on HashiCorp's pricing page).
- [x] LICENSE (Apache-2.0) and NOTICE added to `learn/tf-associate`.
- [x] Architecture `CLAUDE.md` written (replaces the Phase 0 one).

### Phase 1 — Skeleton and plan mode
- [~] Copy the CLI shell from k8slings: done for cobra commands, workspace, `update`, markers, progress file (`.tflings.json`, with a reset counter), plain `watch` and quiz mode (`iac1` proves the loop and the "broken fails, solution passes" test). Bubble Tea full-screen `watch` built (`internal/tui`, with tests and a pty run).
- [~] Terraform runner: isolated env, per-exercise `TF_DATA_DIR`, plugin cache mirror config, process-group timeouts, output caps, the provider/backend safety check and the Engine interface are built and tested (`internal/engine`, `internal/runner`). Still to do: `hc-install` fallback, wiring into plan mode.
- [x] Floci manager (`aws up|down|status|reset`, pinned digest, hardened run, replaces a container from another image, keeps the data volume) and `doctor` (Terraform, Docker, Floci, plugin cache with `--prefetch`, `<account>.localhost` with the exact `/etc/hosts` lines). The automatic global reset of decision 5 is not wired yet; only `aws reset` (with confirmation) exists.
- [~] Plan mode + `check.yaml` (plan, config, fmt, variants): built and proven for 7 provider-free exercises (`hcl1`, `fmt1`, `variables1-2`, `outputs1`, `types1`, `expressions1`, `functions1`; the quiz `iac1` is the eighth). Still to do: `workflow` topic, many more exercises, plan exercises that use the AWS provider (need Floci).
- [~] CI (`.github/workflows/ci.yml`, actions pinned by SHA, dependabot): jobs lint, test (Linux and macOS, Terraform 1.12 and latest), aws (Floci in Docker, provider cache, Terraform 1.12 and latest), offline (provider-free proofs with the network off) and windows (informational). **First GitHub run (PR #1, 2026-11-14): every job green**, including the Docker/Floci job on Terraform 1.12 and latest, the offline job (`sudo unshare --net` works on `ubuntu-latest`), macOS, and Windows (informational job; unit tests and provider-free proofs pass there on Terraform 1.12, the AWS-provider exercises are not run there). The exercise lint (`TestExerciseLint`) runs in every job.

### Phase 2 — Apply mode
- [~] Apply mode: built (`internal/exercises/apply.go`): plan checks, then apply only if they pass, then `config` and `state` checks, then `empty_plan_after`; state is kept in the exercise's scratch folder so saves apply incrementally; a save with nothing to change skips the apply (4 s). Proven by `references1` (the real `NoSuchBucket` error for the broken file). Decisions taken 2026-11-14: typed `aws:` checks (not generic reflection), apply runs on every save once the plan checks pass, topics start with resources, references, iteration, data-sources. **Typed `aws:` checks built** (`internal/awscheck`: s3 bucket/versioning/object/list/tags, ssm parameter, dynamodb table, ec2 vpcs/subnets/security groups, iam role, secrets manager secret; read-only allowlist, client built by hand so the learner's AWS environment is never read; proven against Floci; `resources2` uses one). Still to do: the automatic global Floci reset, the topics.
- [~] Topics: 14 exercises done on top of `resources1`: `resources2-3`, `references1-3`, `iteration1-3`, `data-sources1`, `dependencies1` and `lifecycle1-4` (create_before_destroy, prevent_destroy, ignore_changes, replace_triggered_by; provider-free, proven on Terraform 1.12 and latest). Apply exercises prove themselves against Floci in CI. Still to do: more exercises in these topics, then `providers`, `versions`, `conditions`, `sensitive`.

### Phase 3 — State and fix mode
- [x] Fix mode: seeding (`apply` steps and allowlisted Floci `mutate` steps), `tflings tf` and `tflings env`, `run` checks the end state, `reset` re-seeds. Built on 2026-11-14 with `refactoring1`, `drift1`, `drift2`. Not seeded yet: a tfstate written by hand, a stuck lock.
- [ ] Topics: `state-basics`, `backends`, `locking`, `drift` (refresh-only), `refactoring` (moved/removed/state mv), `import` (import blocks, `-generate-config-out`), `debugging`.
- [ ] cli mode and the `inspect` topic.

### Phase 4 — Modules
- [ ] Local modules, scope, outputs between modules; sources and versions with a local git repo; registry specifics via quiz unless a local registry stub proves cheap.
- [ ] Topics: `modules`, `module-versions`; capstone.

### Phase 5 — Exam practice and HCP
- [ ] Quiz mode, question bank (target ~150, all objectives, explanation + docs link each).
- [x] `tflings exam` (first slice: 35 questions). [ ] `tflings progress`, a bigger bank.
- [ ] Optional `hcp` track (`requires = ["hcp"]`, skipped without `TF_TOKEN_app_terraform_io`).

### Phase 6 — Playground
- [ ] `tflings sandbox`, `tflings console`, exercises with EC2 instances that really run (user data, security group ports).

### Phase 7 — Release
- [ ] GoReleaser, Homebrew tap (`cleonte/tap/tflings`), README with demo GIFs, CONTRIBUTING, version watch for Terraform, the AWS provider and Floci.

### Phase 8 — OpenTofu
- [ ] `tofu` engine, `--engine`, `doctor`, `engines` filtering, CI matrix, small `tofu-only` topic.

## Risks

- **Offline resolution of `<account>.localhost`** (accepted, decision 10): with the split endpoints above, the only name still needed is `<account-id>.localhost`, for S3 Control (spike 04: works online everywhere tested; offline on Linux it works only with `127.0.0.1 <account>.localhost` in `/etc/hosts`, which needs root). `doctor` must detect it and print the exact lines. Windows, macOS offline, amd64 and systemd-resolved are untested.
- **Floci moves fast**: a new release can change behaviour. Pin the version, run the full exercise suite before bumping, keep the bump a deliberate PR (version watch opens it).
- **Emulator gaps**: an AWS feature Floci gets wrong would teach the wrong thing. Use only resource types in Floci's Terraform compatibility suite or proven by our own CI; prefer S3, VPC, IAM, SSM, Secrets Manager, DynamoDB, Route 53, KMS.
- **Docker is required** for all hands-on exercises. Acceptable (k8slings learners have it already); `doctor` checks it and the Lima/Colima specifics, and companies that cannot use Docker Desktop can use Colima or OrbStack.
- **First-run download size** (AWS provider + Terraform + Floci): `doctor --prefetch` and a clear progress message.
