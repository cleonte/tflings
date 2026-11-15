# inspect2: read an output from the state

> **Ticket.** The deploy job printed a sensitive output called `api_key` and the
> log was cleaned. The value is still in the state. Read it, then put it in the
> output `answer` of `main.tf`.

```
tflings tf inspect2 output
tflings tf inspect2 output -raw api_key
```

A sensitive output is shown as `<sensitive>` in a list; asking for it by name
prints it. Edit `main.tf`, then check:

```
tflings run inspect2
```
