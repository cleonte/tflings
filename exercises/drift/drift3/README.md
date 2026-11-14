# drift3: someone deleted it

> **Ticket.** The message of the day parameter is gone: somebody deleted it in
> the console. The configuration in `main.tf` is right. Make the parameter exist
> again, and check what Terraform says about it on the way.

```
tflings tf drift3 plan
tflings tf drift3 state list
```

Read the top of the plan: Terraform tells you what changed outside of it. Then
repair the situation and check:

```
tflings run drift3
```
