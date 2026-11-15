# cli3: one command

Write **one** Terraform command in `answer.sh`. tflings reads it and checks it; it does not run it.

> **Task.** Someone changed a tag in the cloud console. Show how the **state** would change to match the real infrastructure, without proposing any change to the infrastructure.

Put the command on one line below the comment (lines that start with `#` are comments).
It starts with `terraform`. No pipes, redirects or `&&`: it is one command.
Then remove the `# I AM NOT DONE` line and run `tflings run cli3`.
