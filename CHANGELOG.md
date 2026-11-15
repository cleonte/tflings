# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/cleonte/tflings/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/cleonte/tflings/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/cleonte/tflings/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cleonte/tflings/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cleonte/tflings/releases/tag/v0.1.0
