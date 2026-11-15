# cli8: one command

Write **one** Terraform command in `answer.sh`. tflings reads it and checks it; it does not run it.

> **Task.** A bucket called `legacy-logs` was created by hand. The configuration already has `resource "aws_s3_bucket" "logs"`. Bring the bucket under Terraform's management with the command (not an import block).

Put the command on one line below the comment (lines that start with `#` are comments).
It starts with `terraform`. No pipes, redirects or `&&`: it is one command.
Then remove the `# I AM NOT DONE` line and run `tflings run cli8`.
