# inspect1: what does the state know?

> **Ticket.** Somebody applied a configuration with an extra resource and then
> lost that file. The state still has it. Find the resource that is in the state
> but not in `main.tf`, read its `input`, and put the text in the output
> `answer` of `main.tf`.

```
tflings tf inspect1 state list
tflings tf inspect1 state show terraform_data.ADDRESS
```

`state list` shows every address; `state show` prints the attributes of one.
Neither changes anything. Edit `main.tf`, then check:

```
tflings run inspect1
```
