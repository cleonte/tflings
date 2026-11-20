# module-versions3: a module that pins Terraform

Modules have versions too, in two places.

| Where | Constraint on | Written as |
|---|---|---|
| `module` block, `version = "~> 5.0"` | the **version of the module** to install | only for modules from a registry; Terraform refuses `version` next to a local `source` such as `./modules/app` (a local folder has no versions) |
| `terraform { required_version = "..." }` inside the module | the **Terraform CLI** that may use the module | any module, local ones included |

The operators are the same as everywhere: `=`, `!=`, `>`, `>=`, `<`, `<=` and `~>`. `~> 5.0` means 5.0 and newer, below 6.0; `>= 1.12` has no upper limit.

This module is local, so it has no `version` to set, but its author pinned the
Terraform version with `= 1.0.0`. `terraform init` stops with
"Unsupported Terraform Core version", naming the module. Read the error, then
fix the constraint **in the module** (`modules/app/main.tf`), not in the root
configuration: it must allow Terraform 1.12 and every newer release.

Docs: https://developer.hashicorp.com/terraform/language/block/module#version and https://developer.hashicorp.com/terraform/language/expressions/version-constraints
