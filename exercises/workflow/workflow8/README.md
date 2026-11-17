# workflow8: validate before you plan

> **Ticket.** Someone edited `main.tf` and now nothing works. Before you plan
> anything, ask Terraform whether the configuration is even valid. Fix every error
> it reports.

```
tflings tf workflow8 validate
```

`terraform validate` reads the configuration only: it checks syntax, argument
names, types and references (it needs `init`, which tflings runs for you, but it
never contacts a cloud and never reads the state). It reports all the errors it
finds, not only the first. What it cannot know is what the real infrastructure
looks like, so a configuration can be valid and still fail at apply.

Fix the file, run `validate` until it says `Success!`, then check:

```
tflings run workflow8
```

Related tutorial: https://developer.hashicorp.com/terraform/tutorials/cli/init
