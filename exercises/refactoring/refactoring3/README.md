# refactoring3: into a module

> **Ticket.** The server `terraform_data.web` used to be written straight in
> `main.tf`. Someone moved its code into the module `modules/web` and called the
> module, and now the plan wants to destroy the old server and build a new one.
> It is the same server. Record the move so that nothing is rebuilt, in the
> configuration, for everyone who shares it.

```
tflings tf refactoring3 plan
```

Edit `main.tf`, apply, then check:

```
tflings run refactoring3
```
