# refactoring1: a resource was renamed

> **Ticket.** A teammate renamed `terraform_data.web` to `terraform_data.server`
> in `main.tf` because "web" was confusing. Now `terraform plan` wants to
> destroy the old server and build a new one. It is the same server, and it must
> not be rebuilt.

Terraform tracks objects by their address, the name in the state. A new name in
the code looks like a new resource, and the old name like one to delete.

Look at the situation with the sealed Terraform of this exercise:

```
tflings tf refactoring1 plan
tflings tf refactoring1 state list
```

Repair it **without destroying anything**, then check:

```
tflings run refactoring1
```

Commands to know: `terraform state mv`, and the `moved` block.
