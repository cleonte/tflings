# modules2: module outputs

What happens in a module stays in the module. The caller can only read what the
module **exports** with an `output` block:

```hcl
# modules/app/outputs.tf
output "name" {
  value = terraform_data.this.input
}

# main.tf
output "app_name" {
  value = module.app.name
}
```

`module.app.terraform_data.this` does not exist for the caller: that is the
point of a module.
