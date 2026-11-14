# modules1: calling a module

A module is a folder of Terraform files. The configuration you run is the
**root module**; it calls others with a `module` block:

```hcl
module "app" {
  source = "./modules/app"
  name   = var.name        # an input: a variable declared inside the module
}
```

The variables of the module are its inputs, and a variable without a default is
required. Its resources get the address of the module in front:
`module.app.terraform_data.this`.
