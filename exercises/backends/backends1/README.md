# backends1: move the state to S3

> **Ticket.** This project keeps its state in a file on one laptop, and the rest
> of the team cannot run it. Move the state to the S3 bucket `tflings-state`
> (key `app/terraform.tfstate`, region `us-east-1`) and turn on locking with a
> lock file. The existing resources must not be rebuilt.

```
tflings tf backends1 state list
```

Edit `main.tf`, then run Terraform's own init (`tflings tf backends1 init ...`),
and check:

```
tflings run backends1
```

Commands to know: the `backend "s3"` block, `terraform init -migrate-state`
(add `-force-copy` to skip the question).
