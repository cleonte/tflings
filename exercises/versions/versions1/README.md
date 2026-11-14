# versions1: required_version

`required_version` in the `terraform` block says which Terraform CLI versions
may run the configuration. Anything else stops with an error before a plan.

| Constraint | Allows |
|---|---|
| `= 1.5.0` (or `1.5.0`) | only 1.5.0 |
| `>= 1.12` | 1.12 and anything newer, including 2.0 |
| `~> 1.12` | 1.12 up to, but not including, 2.0 |
| `~> 1.12.0` | 1.12.x only |

Fix the constraint so Terraform 1.12 and newer 1.x releases can run it.
