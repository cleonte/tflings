# Contributing

tflings is pre-release and moves fast. Issues and pull requests are welcome;
for anything larger than an exercise or a fix, open an issue first so we can
agree on the shape.

## Build and test

```bash
make build        # bin/tflings
make test         # unit tests
make ci           # vet, then every exercise proven on each Terraform in TF_BINS
```

`make ci` needs Terraform 1.12 and the latest in `TF_BINS`. The exercises that
use the AWS provider also need Floci running (`tflings aws up`) and a home with
the provider cached (`TFLINGS_HOME=... tflings doctor --prefetch`, then
`TFLINGS_TEST_HOME=...`); without them those exercises are skipped.

## Adding an exercise

Read [CLAUDE.md](CLAUDE.md) first: it is the design. In short, an exercise is a
folder under `exercises/<topic>/<name>/` with:

- `main.tf` with `# I AM NOT DONE` on line 1 and a `# TODO:` at the fix site
  (fix-mode exercises have none: they are repaired with commands),
- `README.md`: the lesson, with a link to the matching HashiCorp tutorial,
- `check.yaml`: [CEL](https://cel.dev) assertions over the plan, the
  configuration, the state or Floci. Add `variants` so that hard-coding the
  answer fails,
- a solution under `solutions/<topic>/<name>/`,
- an entry in `info.toml` with the exam objective it practises.

`TestExerciseLint` enforces the rules (for example `force_destroy = true` on
every bucket) and `TestEveryExerciseIsProven` requires that the exercise as
shipped fails and the solution passes, so a new exercise cannot be merged
half-built.

## Rules that matter

- **Never touch a real cloud.** Do not weaken the runner's environment or the
  provider check in `internal/runner`.
- Only `internal/engine` may run a `terraform` binary.
- Say only what you ran: facts go in a `RESULTS.md` with the date and versions.
- Conventional commit messages (`feat:`, `fix:`, `docs:`, `test:`).

## Releasing

Pushing a tag `vX.Y.Z` runs the release workflow, which builds the archives (Linux, macOS and Windows, on amd64 and arm64) with [GoReleaser](https://goreleaser.com) and publishes the release. The body of the release is the section of `CHANGELOG.md` for that version (`scripts/release-notes.sh vX.Y.Z` prints it, and fails when there is none, in which case GoReleaser writes the notes from the commits).

So before tagging:

1. Move what is under "Unreleased" in `CHANGELOG.md` into a section `## [X.Y.Z] - date`, and update the links at the bottom.
2. Check the release locally: `goreleaser check`, then `goreleaser release --snapshot --clean --skip=publish` builds every archive into `dist/` without publishing anything.
3. Tag and push: `git tag -a vX.Y.Z -m "..." && git push origin vX.Y.Z`.

The release also updates the Homebrew cask in the tap [cleonte/homebrew-tap](https://github.com/cleonte/homebrew-tap) (`homebrew_casks` in `.goreleaser.yaml`). That needs the repository secret `HOMEBREW_TAP_TOKEN`: a fine-grained personal access token with **Contents: read and write on the tap repository only** (the `GITHUB_TOKEN` of a workflow cannot push to another repository). Without it the release fails at the publish step, after the archives are built.
