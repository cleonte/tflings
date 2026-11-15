# cli2: one command

Write **one** Terraform command in `answer.sh`. tflings reads it and checks it; it does not run it.

> **Task.** Show what would change, and save the plan in a file called `tfplan`, so that it can be applied later.

Put the command on one line below the comment (lines that start with `#` are comments).
It starts with `terraform`. No pipes, redirects or `&&`: it is one command.
Then remove the `# I AM NOT DONE` line and run `tflings run cli2`.
