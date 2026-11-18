# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Badges under the title of the README: CI status, latest release, licence (Apache-2.0) and Terraform 1.12+.
- README badge for the Go version (read from `go.mod`), and the Go and Terraform logos on their badges.
- A weekly `versions` workflow (and `scripts/check-versions.sh`) that reports when a newer Floci, AWS provider or Terraform exists than the pinned ones. It only reports.
- `SECURITY.md` says that release binaries are unsigned and that the Homebrew cask removes the macOS quarantine flag.
- 16 new mock exam questions (`questions/batch5.toml`) for the thinnest objectives: 3f (destroy), 3g (formatting), 4a (resource and data blocks) and 4b (references). The bank has 183 questions.
- Two hands-on exercises for objectives that were quiz only: `workflow7` (`terraform init` and a module it cannot install, plan mode) and `workflow8` (`terraform validate`, fix mode). 97 exercises.

### Changed

- The release workflow runs `go test ./...` (on Terraform 1.12, without Docker) before GoReleaser publishes, and the `aws` CI job runs its tests with `-timeout 60m`.
- Two exercises folders on one machine no longer share state. `tflings init` now gives the folder a random salt (in `.tflings.json`), which names its scratch folders under tflings's home and goes into its fake AWS accounts, so a `reset` in one folder cannot delete the other's saved state. Folders made by an older version have no salt and keep working exactly as before.
- `run` now records a pass in the progress file of the exercises folder it works on, the same file the reset counter lives in, instead of always in the current directory.
- The safety check that refuses a configuration which could leave the fake AWS now also reads backend configuration files (`*.tfbackend`, `*.hcl`) and refuses `assume_role` written as an argument of the s3 backend. `tflings tf` refuses `-backend-config` values that set credentials, a role or an endpoint other than Floci (or name a file that is not one of the exercise's own), `-var` values that give an endpoint another URL, and a `-var-file` outside the exercise's folder.

### Fixed

- `workflow6` could not be done as its README says: every `tflings tf` call cleared the exercise's scratch folder, so the saved plan of `tflings tf workflow6 plan -out=tfplan` was gone before `show` and `apply tfplan`. tflings now removes only the files it wrote itself; a saved plan or a file Terraform generated stays. The proof of a fix exercise's solution now runs each command the way `tflings tf` does, so a mistake like this is caught.

- Two tflings processes (a `watch` in one terminal, `tflings tf` or `run` in another) could use the same exercise's scratch folder at once, and one could delete the folder under the other (the likely cause of a "chdir .../runs/.../src" error). The folder now has a lock that is held for the whole command, `reset` waits for it (or says it could not and changes nothing), and a fix exercise is built in a side folder and moved into place when it is whole.

## [0.9.2] - 2026-11-15

### Fixed

- `tflings update` now also replaces the `info.toml` entry (objective, hint, link) of exercises you already have when the binary's is newer. Before, only new exercises were added, so a folder made by an older version kept old objective ids and `tflings progress` counted exercises under the wrong objective. `info.toml` holds no progress (that is in `.tflings.json`), and the earlier file is kept as `info.toml.bak`.
- When the setup of a fix exercise fails (for example because an earlier setup of the same exercise, in another folder that shares the Floci container, left a lock or an object behind), the message now says to run `tflings reset <exercise>`, which starts it again in a fresh account. A failed setup also removes its "built" marker.

## [0.9.1] - 2026-11-15

### Added

- Two demo recordings in the README (`misc/demo-tui.gif` and `misc/demo-exam.gif`), made with [vhs](https://github.com/charmbracelet/vhs) from `demo-tui.tape` and `demo-exam.tape`, and a "first five minutes" section.

### Fixed

- In the full-screen view, a long explanation or failure message is wrapped to the width of the terminal instead of being cut off.
- `tflings progress` printed its "run this inside your exercises folder" note to stderr, so it could appear above the table; it is now printed after it.

## [0.9.0] - 2026-11-15

### Added

- The `cli` exercise mode and ten exercises (`cli1` to `cli10`): you write one Terraform command in `answer.sh` for a short task (apply a saved plan, `plan -refresh-only`, `init -migrate-state`, `force-unlock`, `apply -replace`, `import`, `fmt -check -recursive`, ...). tflings reads it and compares it with the accepted form; it is never run, so these need neither Terraform nor Docker. A wrong but tempting command (`taint`, `refresh`, `destroy`) gets a hint that says why. 95 exercises.

## [0.8.0] - 2026-11-15

### Added

- `tflings doctor --install-terraform`: downloads Terraform from releases.hashicorp.com into tflings's own folder (nothing system-wide), with HashiCorp's `hc-install`, which checks the release's SHA256 sums and the signature on them. The default is the newest 1.12.x, the version the exam tests (`--terraform-version 1.12.3` or a constraint chooses another), and tflings uses it before any `terraform` on PATH. `doctor` now says to run it when no Terraform is found. Tried on macOS arm64 and on Ubuntu 26.04 arm64 with no Terraform on PATH.
- Three exercises for topics the exam asks about: `workspaces1` (CLI workspaces: a second state for the same configuration), `iteration7` (`count` against `for_each`: removing an item must not shift the others) and `conditions5` (`can(regex(...))` in a validation): 85 exercises.

### Fixed

- The states of CLI workspaces (`terraform.tfstate.d`) are kept between runs of an apply or fix exercise, like the default state.

## [0.7.0] - 2026-11-15

### Added

- `tflings version` (it only worked as `tflings --version`).
- `tflings exam --review`: asks again, with the answer after each, the questions you missed in your last sitting.
- Exam questions can show a code snippet. 12 new questions use one (what a `for_each` plans, what a failing `validation` does, what `-/+` in a plan will do, variable precedence, an `import` block, provider aliases, a version constraint, and more).
- 12 more questions for the objectives that had one (2b, 3a, 5a, 6a): 167 questions in all.

### Fixed

- A command or flag that does not exist, or any error a command did not report itself, made tflings exit with status 1 and print nothing. It now prints `Error: ...`, and for an unknown command or flag points to `tflings --help` (cobra's "Did you mean" suggestion is kept).

## [0.6.0] - 2026-11-15

### Added

- 77 more exam questions, 143 in all, for every objective of the 004 content list. Batch 2 (38) covers IaC and providers (1, 2), the core workflow (3), resources, data, variables, types, expressions and dependencies (4a to 4f) and modules (5). Batch 3 (39) covers HCP Terraform (8a to 8d: execution modes, agents, run tasks, policy levels, dynamic credentials, speculative plans), drift (6d), the state commands (7b) and more of 2d, 3d, 3e, 4b, 4d, 4e and 5d.

## [0.5.0] - 2026-11-15

### Added

- `tflings progress`: for each of the 37 objectives of the 004 content list, the exercises done and the score on the mock exam questions, with a `!` for what needs work. `tflings exam` now keeps its results (`exam-history.jsonl` in tflings's home folder), and `exam --weak` asks about the three objectives you score lowest on, those never asked first.
- 31 more exam questions (66 in all): conditions, secrets, local and remote backends, locking, drift, import, state commands and logging.
- `tflings exam`: a mock exam from an embedded bank of multiple choice questions (35 so far), including pick-two questions. It asks in random order, can be timed, limited to an objective (`--objective 4g`) or run as practice with the answer after each question, and reports the score per objective with the exercises to redo. HashiCorp does not publish the real exam's question count or pass mark, so the defaults (30 questions, 60 minutes) are ours.

## [0.4.0] - 2026-11-15

### Added

- Release builds: archives for Linux, macOS and Windows (amd64 and arm64) from GoReleaser, and a Homebrew cask (`brew install cleonte/tap/tflings`).

### Changed

- When `init` finds no AWS provider in the cache, the message says to run `tflings doctor --prefetch`.

## [0.3.0] - 2026-11-15

### Added

- 20 exercises, so that every objective of the Terraform Associate 004 content list (1a to 8d) has at least one: the `workflow` topic (`workflow1` to `workflow6`: `destroy -target`, `-var`, a saved plan, and quizzes on the workflow, `init` and `validate`), `inspect1` and `inspect2` (`state list/show`, `output -raw`), `debugging1` (`TF_LOG`), `hcp1` to `hcp4` (quizzes), `backends0` (the local backend), `modules5` (module sources), `providers3` (passing a provider to a module), and quizzes `iac2`, `iac3`, `providers4`, `state3`.
- A lint that fails when an exercise's `objective` is not in the 004 content list.

### Changed

- The objective of 23 existing exercises was corrected to match the content list (conditions 4g, sensitive data 4h, module versions 5d, and others).

## [0.2.0] - 2026-11-14

### Added

- The `modules` and `module-versions` topics (`modules1` to `modules4`, `module-versions1`, `module-versions2`), `refactoring3` and `data-sources2`: 62 exercises in all.

### Changed

- Each exercise has its own plugin cache, so exercises can run at the same time.

## [0.1.0] - 2026-11-14

### Added

- First tagged version: the CLI (`init`, `list`, `run`, `hint`, `reset`, `solution`, `update`, `verify`, `watch`, `doctor`, `aws up|down|status|reset`, `tf`, `env`), plan, apply, fix and quiz exercises (54), proven on Terraform 1.12 and the latest, against the Floci AWS emulator.

[Unreleased]: https://github.com/cleonte/tflings/compare/v0.9.2...HEAD
[0.9.2]: https://github.com/cleonte/tflings/compare/v0.9.1...v0.9.2
[0.9.1]: https://github.com/cleonte/tflings/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/cleonte/tflings/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/cleonte/tflings/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/cleonte/tflings/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/cleonte/tflings/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/cleonte/tflings/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/cleonte/tflings/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/cleonte/tflings/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cleonte/tflings/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cleonte/tflings/releases/tag/v0.1.0
