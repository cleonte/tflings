# cli5: one command

Write **one** Terraform command in `answer.sh`. tflings reads it and checks it; it does not run it.

> **Task.** The `backend` block was changed from local to S3, and `terraform init` says the backend changed. Run init so that the existing state is **copied** to the new backend.

Put the command on one line below the comment (lines that start with `#` are comments).
It starts with `terraform`. No pipes, redirects or `&&`: it is one command.
Then remove the `# I AM NOT DONE` line and run `tflings run cli5`.
