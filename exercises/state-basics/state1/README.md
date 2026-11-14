# state1: stop managing it, but keep it

> **Ticket.** The `legacy` flag parameter now belongs to another team, and they
> manage it themselves. Remove it from this project, **without deleting it**: it
> is in use right now.

If you only delete the block from `main.tf` and apply, Terraform destroys the
parameter, because it is in the state and no longer in the configuration.

```
tflings tf state1 state list
tflings tf state1 plan
```

Repair the situation, then check:

```
tflings run state1
```

Commands to know: `terraform state list`, `terraform state rm`, and the
`removed` block.
