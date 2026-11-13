# CLAUDE.md — tflings

tflings is "rustlings for Terraform": a Go CLI that teaches Terraform (aimed at the Terraform Associate 004 exam) through small broken exercises, run with the real `hashicorp/aws` provider against **Floci**, a local AWS emulator. It is the sibling of k8slings and golings and copies their shape.

**Status: Phase 0 (spikes).** No Go code yet. Read `ROADMAP.md` first: it is the plan and the list of decisions. The full architecture document replaces this file at the end of Phase 0.

## Where things are

- `ROADMAP.md` — the plan, decisions, phases.
- `spikes/` — Phase 0 experiments. `spikes/README.md` is the brief; each spike writes its findings to `spikes/<nn>-<name>/RESULTS.md`.
- Sibling projects to copy from: `../../k8s/k8slings` or wherever k8slings is cloned (ask), and `~/Work/git/go/golings`.
- Source material for exercises: `../learn/tf-associate/`.

## Rules

- **Never touch real AWS.** Every Terraform or AWS CLI command runs with a clean environment: `env -i` plus only `PATH`, `HOME`, the fake keys, `AWS_ENDPOINT_URL` pointing at the local Floci, `AWS_CONFIG_FILE=/dev/null`, `AWS_SHARED_CREDENTIALS_FILE=/dev/null`. Never use the user's AWS profiles, SSO sessions or `~/.aws`. If a command would need real credentials, stop and ask.
- Floci for tflings runs on **port 4577** (not 4566, which the user may use for something else), storage under the spike folder, never globally.
- Pin versions and write them down (Floci 2.4.0 unless a spike says otherwise; Terraform 1.12.x and latest; AWS provider `~> 6.0`).
- Downloads go under `spikes/.cache/` (git-ignored); verify checksums when the project publishes them.
- Facts you checked go into `RESULTS.md` with the date and the exact versions. Do not write anything as fact that you did not run.
- Commit per spike on a branch (`phase0-spikes`), conventional commit messages, no push unless asked.
