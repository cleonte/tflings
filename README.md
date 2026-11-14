# tflings

Learn Terraform by fixing small broken exercises, like [Rustlings](https://github.com/rust-lang/rustlings), for the **HashiCorp Terraform Associate (004)** exam. It is a sibling of [k8slings](https://github.com/cleonte/k8slings) and [golings](https://github.com/cleonte/golings).

You get a folder of Terraform files with a `# I AM NOT DONE` line and a `# TODO:` where something is wrong. You fix them, save, and tflings tells you what Terraform says and what is still not right. Exercises that build things use the real `hashicorp/aws` provider against [Floci](https://github.com/floci-io/floci), a free local AWS emulator that tflings runs in Docker. Nothing ever talks to a real AWS account.

> **Status: pre-release, expect breakage.** The skeleton, plan mode, apply mode, the full-screen view, `doctor` and 63 exercises work, fix mode among them (`tflings tf <exercise> ...` repairs a situation that has gone wrong). Most of the planned ~120 exercises and the releases do not exist yet. See [ROADMAP.md](ROADMAP.md).

## What you need

- **Terraform 1.12 or newer** (the exam tests 1.12). tflings uses the one on your `PATH`, or the one in `TFLINGS_TERRAFORM`.
- **Docker** (Docker Desktop, OrbStack, Colima, Lima or Docker Engine) for exercises that use the AWS provider. Quizzes and exercises that need no cloud run without it.
- Go 1.26 or newer to build it for now.

`tflings doctor` checks all of this and says how to fix what is missing.

## Try it

There are no release binaries yet. With Go 1.26 or newer:

```bash
go install github.com/cleonte/tflings/cmd/tflings@latest   # puts tflings in $(go env GOPATH)/bin
mkdir ~/tflings-play && cd ~/tflings-play
tflings init            # creates ./tflings-exercises
cd tflings-exercises
tflings doctor          # what works on this machine, and what to fix
tflings watch           # full-screen view: edit a file, save, see the result
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`. To work on tflings itself, clone the repository and run `make build`, which writes `bin/tflings`.

For exercises that use the AWS provider, once:

```bash
tflings aws up              # starts Floci in Docker (pulls about 370 MB the first time)
tflings doctor --prefetch   # downloads the AWS provider (about 250 MB) so exercises start offline
```

## Commands

| Command | What it does |
|---|---|
| `init [dir]` | Write the exercises into a folder. Never overwrites your work. |
| `watch` | Full-screen view that checks the next exercise whenever you save. `--plain` for line mode. |
| `run next` / `run <name>` | Check one exercise. |
| `hint <name>` | Show the hint and the documentation link. |
| `list` | Exercises by topic, with progress. |
| `verify` | Check every exercise in order. |
| `reset <name>` | Put an exercise back as it was. It gets a fresh, empty fake AWS account. |
| `solution <name>` | Show the solution (hidden while the exercise is pending, unless you ask). |
| `tf <name> <terraform args>` | Fix exercises: run Terraform in the exercise's sealed environment (`tflings tf drift1 plan`). |
| `env <name>` | Print that environment, for a shell of your own. |
| `update` | Add the exercises your folder does not have yet. |
| `aws up\|down\|status\|reset` | Manage the local AWS (Floci). |
| `doctor [--prefetch]` | Check this machine. |

## How it works

- **Plan mode** copies your exercise to a scratch folder, runs `terraform init`, `plan` and `show -json`, and checks the result with [CEL](https://cel.dev) expressions that live inside the binary (so they are not a spoiler). Some checks plan again with other variable values, so writing the expected answer by hand does not pass.
- **Terraform runs in a boundary.** It gets an environment built from nothing (your shell's variables are not copied), a wall-clock timeout, a cap on its output, and fake credentials. A check refuses `provider "aws"` and `backend "s3"` blocks that name a profile, keys, a role, or an endpoint other than the local Floci.
- **One fake AWS account per exercise.** Floci uses a 12-digit access key ID as the account, so exercises never see each other's resources.

The design and the experiments behind it are in [CLAUDE.md](CLAUDE.md) and [spikes/](spikes/).

## Known limits

- The `aws` provider needs `<account>.localhost` to resolve (one call of the provider sends to that name). Docker's and most desktop systems' DNS do this; if yours does not, `tflings doctor` prints the exact `/etc/hosts` lines to add.
- Tested on macOS (Apple silicon), Linux arm64 and Linux x86_64 with native Docker, on Terraform 1.12.2 and 1.17.2. Windows ran the exercises that need no AWS in CI; Windows with Docker, and Intel Macs, are untested.
- "Bucket name already taken" cannot be taught with Floci (its S3 names are per account), so those are quiz questions.

## Contributing and security

Issues and pull requests are welcome: see [CONTRIBUTING.md](CONTRIBUTING.md). To report a way out of the sandbox, see [SECURITY.md](SECURITY.md).

## License

Apache-2.0, see [LICENSE](LICENSE) and [NOTICE](NOTICE). Floci is MIT licensed and is downloaded, not included.
