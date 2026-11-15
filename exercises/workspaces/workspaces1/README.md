# workspaces1: one configuration, two states

> **Ticket.** The `app` server runs in production. The team wants a copy for staging,
> built from the **same configuration**, and the production state must not be touched.

A Terraform **workspace** is a separate state for the same configuration (not an HCP
Terraform workspace, which is more). Every configuration starts in the workspace
`default`. The name is available as `terraform.workspace`.

```
tflings tf workspaces1 workspace list
tflings tf workspaces1 workspace new staging
tflings tf workspaces1 plan
```

`workspace new` creates the workspace and switches to it. Build the staging copy, then check:

```
tflings run workspaces1
```
