# cli6: one command

Write **one** Terraform command in `answer.sh`. tflings reads it and checks it; it does not run it.

> **Task.** A pipeline run died and left a lock on the state. The error shows the lock ID `4f3a-91c2`. Nobody is running Terraform. Release the lock.

Put the command on one line below the comment (lines that start with `#` are comments).
It starts with `terraform`. No pipes, redirects or `&&`: it is one command.
Then remove the `# I AM NOT DONE` line and run `tflings run cli6`.
