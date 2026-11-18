# Security

tflings runs Terraform for you, so its main promise is a **safety boundary**: an
exercise never touches a real cloud account. Terraform runs with an environment
built from nothing (fake credentials, the local Floci as the only endpoint), and
tflings refuses configurations that name a profile, real-looking keys, a role,
or an endpoint other than the local Floci.

## Reporting a problem

If you find a way to make tflings reach a real AWS account, read your own
credentials or files, run a command you did not ask for, or escape the boundary
in any other way, please **do not open a public issue**. Use GitHub's private
reporting instead: the **Security** tab of this repository, then **Report a
vulnerability**.

Include the tflings version (`tflings --version`), your OS, and the smallest
exercise or configuration that shows it. This is a project run in spare time, so
there is no guaranteed response time, but a boundary escape is the first thing
that gets fixed.

## Release binaries are not signed

The archives on the GitHub releases page are built by GoReleaser in GitHub
Actions and come with a `checksums.txt`, but neither the binaries nor the
checksums are signed or notarized. The checksums only tell you that the download
is intact, not who built it. On macOS the Homebrew cask (`brew install
cleonte/tap/tflings`) removes the quarantine flag from the binary after
installing it, which is what stops Gatekeeper from refusing an unsigned binary;
if that matters to you, build from source (`go install
github.com/cleonte/tflings/cmd/tflings@<tag>`) instead.

## Supported versions

tflings is pre-release. Only the latest commit of `main` is supported.
