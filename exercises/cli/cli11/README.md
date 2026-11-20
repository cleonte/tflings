# cli11: the first step of the workflow

Write **one** Terraform command in `answer.sh`. tflings reads it and checks it; it does not run it.

> **Task.** You cloned a repository of Terraform files onto a clean machine. Nothing has been run in it yet, so `plan` would fail. Run the first step of the workflow: prepare the working directory (it installs the providers and modules the files need).

The core workflow is write, **init**, plan, apply. `init` is safe to run again at any time.

Put the command on one line below the comment (lines that start with `#` are comments).
It starts with `terraform`. No pipes, redirects or `&&`: it is one command.
Then remove the `# I AM NOT DONE` line and run `tflings run cli11`.

Docs: https://developer.hashicorp.com/terraform/intro/core-workflow
