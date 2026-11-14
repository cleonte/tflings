# drift2: the manual change was right

> **Ticket.** During an incident the on-call engineer changed the message of
> the day to `Maintenance in progress`, and that message must stay. This time
> the configuration is the one that is wrong: make `main.tf` and the state agree
> with what is really there, and do not touch the real parameter.

```
tflings tf drift2 plan
tflings tf drift2 apply -refresh-only
```

Edit `main.tf`, then check:

```
tflings run drift2
```
