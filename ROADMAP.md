# ROADMAP — tflings

tflings is a Rustlings-style course for Terraform, built the way [k8slings](https://github.com/cleonte/k8slings) and [golings](https://github.com/cleonte/golings) are built: same CLI shape, `info.toml` manifest, `exercises/` + `solutions/` trees, `# I AM NOT DONE` marker, Bubble Tea `watch`, `doctor`, CI that proves every exercise ("broken fails, solution passes"). This file is the plan; `CLAUDE.md` (written in Phase 0) will be the architecture.

The syllabus follows the **HashiCorp Terraform Associate (004)** study guide and exam content list. What is new compared with k8slings is the **simulator**: a fake cloud that lives inside the tflings binary, so learners can create, break, drift, import and destroy "infrastructure" offline, for free, with no AWS/Azure account.

## Facts checked while writing this plan (2026-11-13)

- Terraform Associate **004** tests **Terraform 1.12** and includes HCP Terraform content. Prerequisites: basic terminal skills, basic cloud/on-prem architecture. ([study guide](https://developer.hashicorp.com/terraform/tutorials/certification-004/associate-study-004), [exam content list](https://developer.hashicorp.com/terraform/tutorials/certification-004/associate-review-004))
- Current Terraform is 1.16.x (1.17.0 just out). **Decision proposed: floor 1.12 (the exam version), CI on 1.12 and latest.** Features newer than 1.12 stay out of the core path, or are tagged "beyond the exam".
- LocalStack's free Community edition was sunset in March 2026 (account required now). That is one more reason not to build the course on an AWS emulator: **the simulator is our own**.
- To verify in Phase 0: the official exam format (question count, duration, question types) before building the mock exam; the current HCP Terraform free tier limits before building the HCP track.

## The eight objectives → topics

| Obj | Exam objective | tflings topics |
|---|---|---|
| 1 | IaC concepts (1a–1c) | `iac` (quiz) |
| 2 | Fundamentals: providers, versions, multiple providers, state (2a–2d) | `providers`, `versions`, `state-basics` |
| 3 | Core workflow: init, validate, plan, apply, destroy, fmt (3a–3g) | `workflow`, `fmt` |
| 4 | Configuration (4a–4h) | `resources`, `data-sources`, `references`, `variables`, `outputs`, `types`, `expressions`, `functions`, `iteration` (count/for_each/dynamic), `dependencies`, `lifecycle`, `conditions` (validation, pre/postconditions, `check`), `sensitive` (sensitive, ephemeral, write-only, Vault) |
| 5 | Modules (5a–5d) | `modules`, `module-versions` |
| 6 | State management (6a–6d) | `backends`, `locking`, `drift`, `refactoring` (moved, removed, state mv/rm) |
| 7 | Maintain infrastructure (7a–7c) | `import`, `inspect` (state list/show, console, output), `debugging` (TF_LOG) |
| 8 | HCP Terraform (8a–8d) | `hcp` (quiz always; hands-on optional, needs an HCP account) |
| — | Capstone | `capstone`: a broken multi-module stack with drift and a stuck lock, fixed like an incident |

Planning size: **~120 exercises in ~25 topics**, plus a question bank for the mock exam.

## Design

### Exercise modes

Like k8slings, each exercise uses the cheapest mode that can prove the skill.

| Mode | What the learner does | How it is checked | Needs |
|---|---|---|---|
| **plan** | Fix `main.tf` (and friends) | `terraform init` (offline) + `validate` + `plan -out` → `terraform show -json` → CEL assertions over the plan and configuration | `terraform` |
| **apply** | Fix the config so it really builds | Apply against the simulator, then CEL over `show -json` state, outputs and the simulator's inventory | `terraform` |
| **fix** | Repair a seeded broken situation with the CLI (drift, stuck lock, resource renamed in code, resource created by hand that must be imported) | End state of simulator + Terraform state; plan must be empty | `terraform` |
| **cli** | Write one Terraform command in `answer.sh` (`terraform state show ...`, a `console` expression, `output -raw ...`) | Allowlist parser (like k8slings query mode), then compare with the reference answer, with a variant to catch hard-coding | `terraform` |
| **quiz** | Edit `quiz.toml` and set `answer = "b"` (or several) | Compare with the answer key; the explanation is shown after a pass | nothing |

`quiz` covers what cannot be practised hands-on (IaC theory, HCP governance features) and gives exam-style practice. Everything else is hands-on.

Unlike k8slings, almost every mode needs the `terraform` binary. That is a light requirement (one static binary), and `tflings doctor` can install a pinned version into tflings's own cache with `hashicorp/hc-install` if the learner has none. The user's own `terraform` on PATH is used when its version is new enough.

### The simulator ("Tflings Cloud")

A small fake cloud with a real Terraform provider in front of it, both inside the tflings binary.

- **Provider `tflings/tfl`**, written with `terraform-plugin-framework`. The tflings binary serves it when Terraform launches it (argv[0] / subcommand detection), so there is one binary to ship. `init` writes a workspace-local CLI config (`TF_CLI_CONFIG_FILE`) with a `filesystem_mirror` for `tflings/*` and nothing else, so `terraform init` works offline.
- **Resources** (enough to teach every objective, not a clone of any real cloud): `tfl_network`, `tfl_subnet`, `tfl_security_group`, `tfl_instance`, `tfl_bucket`, `tfl_object`, `tfl_dns_record`, `tfl_database`, `tfl_load_balancer`, `tfl_secret`. **Data sources**: `tfl_regions`, `tfl_images`, `tfl_instance` (lookup), `tfl_secret`. **Ephemeral resource**: `tfl_secret` (teaches ephemeral values and write-only attributes).
- **Realistic behaviour**: computed attributes (ids, IPs, ARNs-like `tfl:` ids), attributes that force replacement (so `create_before_destroy` and `-replace` matter), name collisions, region-scoped resources (so provider `alias` matters), quotas, dependency errors ("subnet still in use"), optional latency to make dependency order visible.
- **Several provider versions in the mirror** (e.g. `1.0.0`, `1.4.0`, `2.0.0` with a breaking rename) so version constraints, `.terraform.lock.hcl` and `init -upgrade` are practised offline.
- **A second provider `tflings/dns`** for "multiple providers in one configuration" (2c).
- **Storage**: a JSON file per exercise under `.tflings/cloud/`. Nothing leaves the machine; no real cloud credentials are read (the runner strips `AWS_*`, `ARM_*`, `GOOGLE_*`, `TF_*` from the environment and sets its own `TF_DATA_DIR`).
- **Simulator commands** — the "play" part:
  - `tflings sim ls|show <id>` — look at the cloud like a console would.
  - `tflings sim drift <exercise>` / `sim edit <id> key=value` / `sim delete <id>` / `sim create ...` — change the cloud behind Terraform's back ("someone clicked in the console"), to see drift, refresh-only and import.
  - `tflings sim ui` — a live full-screen view of the cloud (resources appear, change and vanish as `apply` runs). Phase 6 adds `--web`: a local page with the resource graph and plan diff.
- **State backend with locking**: `tflings sim backend` serves Terraform's `http` backend (with `lock_address`/`unlock_address`) on localhost, so remote state, locking, a stuck lock and `force-unlock` are real, not described.
- **Playground**: `tflings sandbox` opens a scratch folder wired to the simulator, with `sim ui` beside it, for free play outside the exercises.

### Checks (`check.yaml`)

Same CEL approach as k8slings, over Terraform's JSON outputs instead of Kubernetes objects:

```yaml
plan:
  - expr: plan.resource_changes.exists(r, r.address == "tfl_instance.web" && "create" in r.change.actions)
    message: "The plan should create tfl_instance.web."
config:
  - expr: config.root_module.variables["region"].default == "eu-west"
    message: "Give the region variable a default."
state:
  - expr: state.values.outputs["web_ip"].value != ""
cloud:
  - expr: cloud.instances.size() == 3
    message: "Three instances should exist in Tflings Cloud."
empty_plan_after: true          # apply/fix: a second plan must have no changes
```

`check.yaml`, `setup/` (seeded state for fix mode) and `solutions/` are embedded and never written to the workspace, as in k8slings.

### Exercise layout

```
exercises/<topic>/<name>/
  README.md        lesson (few lines), links to the matching HashiCorp tutorial
  main.tf ...      the broken configuration; # TODO: at the fix site; # I AM NOT DONE on line 1
  quiz.toml        quiz mode only
  answer.sh        cli mode only
  check.yaml       embedded only
  setup/           fix mode only: seeded simulator state, tfstate, lock (embedded only)
solutions/<topic>/<name>/   the fixed files, or solution.sh for fix mode, plus shortcut-*.sh the check must refuse
```

Every exercise in `info.toml` carries `objective = "4e"` so `tflings list --objective` and the progress report map straight to the exam content list.

### Commands

Same as k8slings (`init`, `watch`, `run`, `hint`, `list`, `verify`, `reset`, `solution`, `update`, `doctor`), with `cluster` replaced by `sim`, plus:

- `tflings terraform ...` — pass-through to Terraform with the exercise's isolated env and CLI config (the only terraform a learner needs).
- `tflings exam [--minutes N]` — timed mock exam drawn from the question bank, scored per objective, with explanations at the end.
- `tflings progress` — readiness per exam objective.

## Phases

### Phase 0 — Decisions and spikes
- [ ] Spike: serve the provider from the tflings binary through a filesystem mirror; `init` fully offline on macOS, Linux, Windows.
- [ ] Spike: multiple provider versions in the mirror and lock-file checksums (`h1:` hashes generated at build time).
- [ ] Spike: CEL over `terraform show -json` (plan and state) on Terraform 1.12 and latest; note the format differences.
- [ ] Spike: `http` backend with locking served by tflings.
- [x] Decided: Terraform first behind an `Engine` interface; OpenTofu in Phase 8.
- [ ] Add a LICENSE to `learn/tf-associate` (or note in `NOTICE`); map each day's lab to exercises.
- [ ] Verify the official exam format and the HCP free tier; write `CLAUDE.md`.

### Phase 1 — Skeleton and plan mode
- [ ] Copy the CLI shell from k8slings (cobra, Bubble Tea v2, workspace, `update`, markers, progress file), rename, strip Kubernetes.
- [ ] Terraform runner: isolated env, per-exercise `TF_DATA_DIR`, timeouts, output caps, `hc-install` fallback.
- [ ] Plan mode + `check.yaml` (plan/config).
- [ ] Topics: `workflow`, `fmt`, `variables`, `outputs`, `types`, `expressions`, `functions` (using only `terraform_data` and the simulator's data sources).
- [ ] CI: every exercise proven on Terraform 1.12 and latest.

### Phase 2 — Simulator v1 and apply mode
- [ ] `tflings/tfl` provider, cloud store, `sim ls/show/edit/delete`.
- [ ] Apply mode with `cloud` and `state` checks, `empty_plan_after`.
- [ ] Topics: `providers`, `versions`, `resources`, `data-sources`, `references`, `iteration`, `dependencies`, `lifecycle`, `conditions`, `sensitive`.

### Phase 3 — State and fix mode
- [ ] Seeding (simulator state + tfstate + lock), `reset` re-seeds, done is kept.
- [ ] `sim backend` (http, locking), `sim drift`.
- [ ] Topics: `state-basics`, `backends`, `locking`, `drift` (refresh-only), `refactoring` (moved/removed/state mv), `import` (import blocks, `-generate-config-out`), `debugging`.
- [ ] cli mode and the `inspect` topic.

### Phase 4 — Modules
- [ ] Local modules, module scope, outputs between modules; sources and versions with a local git repo and a tiny module registry stub if the spike allows (otherwise quiz for registry specifics).
- [ ] Topics: `modules`, `module-versions`; capstone.

### Phase 5 — Exam practice and HCP
- [ ] Quiz mode, question bank (target ~150 questions, all eight objectives, each with an explanation and a docs link).
- [ ] `tflings exam` and `tflings progress`.
- [ ] Optional `hcp` track (`requires = ["hcp"]`, skipped without `TF_TOKEN_app_terraform_io`): `cloud` block, workspaces and projects, variable sets, run triggers. Optional `vault` add-on (Vault dev server) for objective 4h.

### Phase 6 — Simulator UI and playground
- [ ] `sim ui` live TUI; `tflings sandbox`.
- [ ] `sim ui --web`: local page with the dependency graph, plan diff and the cloud inventory.

### Phase 7 — Release
- [ ] GoReleaser, Homebrew tap (`cleonte/tap/tflings`), README with demo GIFs, CONTRIBUTING, version watch for Terraform releases.

### Phase 8 — OpenTofu
- [ ] `tofu` engine, `--engine` flag and `doctor` support, `engines` filtering, CI matrix, small `tofu-only` topic.

## Decisions (2026-11-13)

1. **Simulator only.** No real AWS/Azure track, ever: nothing in tflings reads cloud credentials or talks to a real cloud. (The optional HCP track in Phase 5 is the one exception that needs an account; it is skippable and stays last.)
2. **Terraform first, OpenTofu-ready.** The runner talks to an `Engine` interface from day one; `tofu` support is Phase 8 (see below).
3. **Repo `github.com/cleonte/tflings`, Apache-2.0, Go floor 1.26**, like golings and k8slings.
4. **Source material: `~/Work/git/terraform/learn/tf-associate`** (see below).

## Source material: `learn/tf-associate`

A 12-day plan (day 0 to day 12) already mapped to the 004 objectives, written for no cloud account (`local`, `random`, `terraform_data`). It is Cristian's own; it has no LICENSE file, so add one there (or state in `NOTICE` that it is the author's own) before adapting.

| Source | Becomes |
|---|---|
| `day-00-hcl-basics` | `hcl` intro topic (blocks, types, expressions), plan mode |
| `day-01-workflow` | `iac` quiz + `workflow`, `fmt` |
| `day-02-providers` | `providers`, `versions` (the simulator's multi-version mirror replaces `random`) |
| `day-03-state` | `state-basics`, first `drift` exercises |
| `day-04-resources-dependencies` | `resources`, `data-sources`, `references`, `dependencies`, `lifecycle` |
| `day-05-variables` | `variables`, `outputs`, `types` |
| `day-06-expressions` | `expressions`, `functions`, `iteration` |
| `day-07-conditions-secrets` | `conditions`, `sensitive` (ephemeral and write-only via `tfl_secret`) |
| `day-08-modules` | `modules`, `module-versions` |
| `day-09-backends-state-locking` | `backends`, `locking` (via `sim backend`) |
| `day-10-maintain` | `import`, `refactoring`, `inspect`, `debugging` |
| `day-11-hcp-terraform` | `hcp` quiz (+ optional hands-on track) |
| `day-12-mock-exam`, `practice-questions.md` | seed of the question bank (30 questions). Note: their key is almost all "b"; the quiz loader must shuffle options. |
| `extras/cheatsheet`, `glossary` | hint texts and `tflings hint` docs links |

The days' labs use `local`/`random`; each lab is rewritten against the simulator so it can be checked (apply mode) and broken (fix mode).

## OpenTofu (Phase 8)

How hard: **moderate, about 10–15% extra**, if the runner is engine-agnostic from the start. What carries over unchanged: the provider (same plugin protocol), the filesystem mirror, `show -json` output, CEL checks, most HCL. What costs work:

- **Engine abstraction**: binary name, version detection, install (`hc-install` for Terraform, OpenTofu's own release downloads), env vars (`TF_*` mostly shared).
- **Divergent features**: OpenTofu has state encryption, early variable evaluation in backend/module sources, provider `for_each`, `-exclude`; Terraform has HCP Terraform / `cloud` block and its own ephemeral/write-only timeline. Exercises get `engines = ["terraform", "tofu"]` (default both); HCP and exam-specific ones are `terraform` only, and a small `tofu-only` topic can show the extras.
- **CI matrix doubles** (Terraform 1.12, Terraform latest, OpenTofu latest).
- **Wording**: lessons say "Terraform" because the exam does; with `--engine tofu` the hints and commands print `tofu`.

So: build Terraform-only through Phase 7, but never call `terraform` outside the `Engine` interface, and keep `engines` in `info.toml` from Phase 1 (always `["terraform"]` until Phase 8).
