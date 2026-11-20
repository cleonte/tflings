# modules6: a local source is relative to the module that writes it

A module can call other modules. A local `source` (`./...` or `../...`) is a path
from the folder of the file that contains the `module` block, not from where you
run Terraform:

```
main.tf                 module "app" { source = "./modules/app" }
modules/app/main.tf     module "db"  { source = "../db" }     # modules/db
modules/db/main.tf
```

Here `modules/app` calls `db` with the path it would have in the root folder,
and `terraform init` cannot find it. Read the error, then fix the `source` in
`modules/app/main.tf` so that the nested module is found and planned.

Docs: https://developer.hashicorp.com/terraform/language/modules/sources#local-paths
