# conditions1: variable validation

A `validation` block inside a `variable` checks the value before Terraform
plans anything. If `condition` is false, Terraform stops and prints
`error_message`.

```hcl
variable "port" {
  type = number

  validation {
    condition     = var.port >= 1 && var.port <= 65535
    error_message = "The port must be between 1 and 65535."
  }
}
```

tflings plans your configuration with good and bad ports and expects the bad
ones to be refused.
