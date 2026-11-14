# state2: build it again

> **Ticket.** The app server is misbehaving and a restart did not help. The
> configuration is correct and has not changed, so `terraform plan` says there
> is nothing to do. Make Terraform destroy the server and build a new one.

```
tflings tf state2 plan
tflings tf state2 state show terraform_data.app
```

Note the `id` of the server, repair the situation, then check:

```
tflings run state2
```

Commands to know: `terraform apply -replace=ADDRESS` and the older
`terraform taint`.
