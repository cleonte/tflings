# import1: a bucket nobody told Terraform about

> **Ticket.** A bucket called `tflings-legacy` was created by hand a long time
> ago, and it has a file in it that must not be lost. `main.tf` already describes
> it, but Terraform has never heard of it. Bring the bucket under Terraform's
> management **without recreating it**.

```
tflings tf import1 plan
```

Read what the plan wants to do. Then repair the situation, and check:

```
tflings run import1
```

Two ways exist: the `terraform import` command, and an `import` block in the
configuration. Either is fine here.
