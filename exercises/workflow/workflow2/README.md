# workflow2: apply with a different value, once

> **Ticket.** The `app` server is `small` by default. For tonight's load test it
> must be `large`, and nobody wants to edit the configuration for one night. Make
> the server `large`.

A variable can be set for one run on the command line:

```
tflings tf workflow2 plan -var tier=large
```

Plan first, read it, then apply the same way. Then check:

```
tflings run workflow2
```

(The next plain `plan` will want to go back to `small`: values set with `-var`
are not remembered. A `terraform.tfvars` file or `TF_VAR_tier` is how you keep one.)
