# modules4: a module has its own scope

A module sees nothing of its caller. Variables, locals and resources of the root
module do not exist inside it, even though the files sit next to each other. The
only way in is an **input**: a `variable` declared in the module and set in the
`module` block.

```hcl
# modules/app/main.tf
variable "environment" { type = string }

# main.tf
module "app" {
  source      = "./modules/app"
  environment = var.environment
}
```
