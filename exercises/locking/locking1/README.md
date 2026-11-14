# locking1: the lock that never went away

> **Ticket.** Every Terraform command on this project fails: it says the state
> is locked. The pipeline run that held the lock (`ci@build-agent-7`) died last
> night, and nobody is running Terraform now. Make the project usable again,
> without losing the state.

The state of this project lives in an S3 bucket, and the lock is a file next to
it (`use_lockfile = true`).

```
tflings tf locking1 plan
```

Read the whole error. Then repair the situation, and check:

```
tflings run locking1
```
