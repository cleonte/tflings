# modules5: where a module comes from

`source` tells Terraform where to get a module. Terraform decides by the text:

| `source` | Where it comes from |
|---|---|
| `./modules/app`, `../shared` | a local folder (must start with `./` or `../`) |
| `hashicorp/consul/aws` | the public registry (`namespace/name/provider`) |
| `app.terraform.io/acme/vpc/aws` | a private registry |
| `github.com/acme/app`, `git::https://...` | Git |

The module of this exercise is in the folder `modules/app`, next to `main.tf`.
`init` fails. Read the error and fix the `source`.
