# drift1: someone changed it by hand

> **Ticket.** The message of the day was changed in the console, and now it says
> something it should not. The configuration in `main.tf` is right. Bring the
> real parameter back to what the configuration says.

```
tflings tf drift1 plan
```

Terraform finds the difference by itself, because a plan refreshes first. Repair
the situation, then check:

```
tflings run drift1
```
