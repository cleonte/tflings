# cli7: one command

Write **one** Terraform command in `answer.sh`. tflings reads it and checks it; it does not run it.

> **Task.** The server `aws_instance.web` is misbehaving. Nothing in the configuration changed. Make Terraform destroy and build it again in the next apply, without editing any file.

Put the command on one line below the comment (lines that start with `#` are comments).
It starts with `terraform`. No pipes, redirects or `&&`: it is one command.
Then remove the `# I AM NOT DONE` line and run `tflings run cli7`.
