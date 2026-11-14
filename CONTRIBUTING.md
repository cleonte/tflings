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
