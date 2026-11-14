# modules3: count and for_each on a module

`count` and `for_each` work on a `module` block as they do on a resource, and
build one copy of the whole module for each element:

```hcl
module "app" {
  source   = "./modules/app"
  for_each = toset(var.apps)
  name     = each.key
}
```

The addresses become `module.app["web"].terraform_data.this`.
