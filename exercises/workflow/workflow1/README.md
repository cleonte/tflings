# workflow1: destroy only what you must

> **Ticket.** The `scratch` environment was only for a demo and costs money. Destroy
> it. The `app` server must keep running.

`terraform destroy` removes everything in the state. To remove one object, either
delete its block and apply, or aim the command at it with `-target`:

```
tflings tf workflow1 state list
tflings tf workflow1 destroy -target=terraform_data.scratch
```

`-target` is meant for exceptions (the plan warns you), because the rest of the
configuration is not looked at. Then check:

```
tflings run workflow1
```
