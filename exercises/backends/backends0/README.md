# backends0: the local backend

When there is no `backend` block, Terraform uses the **local** backend: the state is
the file `terraform.tfstate` in the working directory. The block makes it explicit and
lets you move the file:

```hcl
terraform {
  backend "local" {
    path = "relative/path/to/terraform.tfstate"
  }
}
```

The team keeps all state files under a `state/` folder. Make this configuration
write its state to `state/app.tfstate`.
