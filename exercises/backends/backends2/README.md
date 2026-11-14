# backends2: the key changed

> **Ticket.** A teammate renamed the state key in `main.tf` from
> `app/terraform.tfstate` to `orders/terraform.tfstate` to match the project's
> new name. Now Terraform refuses to run. The state and its resources must end
> up under the new key, and nothing may be rebuilt.

```
tflings tf backends2 plan
```

Read the error. Two init flags look like the cure, and only one of them is:
`-reconfigure` and `-migrate-state`. Repair the situation, then check:

```
tflings run backends2
```
