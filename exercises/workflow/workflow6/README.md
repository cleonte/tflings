# workflow6: review the plan, then apply exactly that plan

> **Ticket.** `main.tf` was changed to give the `app` server a new input, and the
> change has been approved. Apply it, but safely: save the plan, read it, and apply
> **that** plan, so that what runs is what was reviewed even if the configuration or
> the real infrastructure changes in between.

```
tflings tf workflow6 plan -out=tfplan
tflings tf workflow6 show tfplan
tflings tf workflow6 apply tfplan
```

`apply` with a plan file does not ask for confirmation and does not plan again. Then check:

```
tflings run workflow6
```
