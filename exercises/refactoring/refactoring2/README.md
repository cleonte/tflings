# refactoring2: the same rename, for everyone

> **Ticket.** Same story as `refactoring1`: `terraform_data.web` was renamed to
> `terraform_data.server` in `main.tf`, and a plan would destroy and rebuild it.
> But this configuration is shared: the CI pipeline and two teammates have their
> own state. Running `state mv` on your machine fixes only your state. Record the
> rename in the configuration instead, so that every state follows it.

```
tflings tf refactoring2 plan
```

Edit `main.tf`, apply, then check:

```
tflings run refactoring2
```
