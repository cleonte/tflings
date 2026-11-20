# backends3: the state file moves

> **Ticket.** The project has always used the default **local** backend: the
> state is the file `terraform.tfstate` next to the configuration. A teammate
> wants the state files under a `state/` folder and added a `backend "local"`
> block with `path = "state/app.tfstate"`. Now Terraform refuses to run. The
> existing state must end up in the new file, and nothing may be rebuilt.

```
tflings tf backends3 plan
```

Read the error. Changing the backend settings means running `init` again, and
`init` can copy the existing state to the new place. Repair the situation, then
check:

```
tflings run backends3
```

Docs: https://developer.hashicorp.com/terraform/language/backend/local
