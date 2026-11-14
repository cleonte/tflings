# sensitive1: sensitive variables

`sensitive = true` on a variable makes Terraform show `(sensitive value)`
instead of the value, everywhere it flows: plan, apply and output.

```hcl
variable "api_key" {
  type      = string
  sensitive = true
}
```

Careful: this only hides the value on screen. It is still stored **in clear text
in the state file**, so the state must be protected as well (a remote backend
with encryption and access control).
