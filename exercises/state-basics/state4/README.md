# state4: read the state

> **Ticket.** A teammate deployed a new `api` from their laptop and never
> committed the change to `main.tf`. `terraform plan` now wants to change the
> `api` resource back to what the file says. The deployed version is the right
> one: the file must catch up with it. Do not apply anything.

Terraform keeps what it applied last in the **state**. A plan compares three
things: the configuration, the state, and (after a refresh) the real objects.
Here the difference is between the configuration and the state, so the state is
where to read the answer:

```
tflings tf state4 plan
tflings tf state4 state list
tflings tf state4 state show terraform_data.api
```

`state list` names every object Terraform tracks, `state show ADDRESS` prints
everything the state holds for one of them. Edit `main.tf` so that the plan has
nothing to change, then check:

```
tflings run state4
```

Applying would make the state say what the file says and bury the deployed
version, so the check refuses it: it wants the state as your teammate left it.

Docs: https://developer.hashicorp.com/terraform/language/state/purpose and
https://developer.hashicorp.com/terraform/cli/commands/state/show
