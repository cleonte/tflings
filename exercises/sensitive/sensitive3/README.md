# sensitive3: ephemeral values and write-only arguments

`sensitive` hides a value on screen, but it still lands in the state. To keep a
secret out of the state completely, use two features (Terraform 1.11 and newer):

- **ephemeral** values (`ephemeral = true` on a variable) exist only during the
  run;
- **write-only arguments** (`value_wo`, with `value_wo_version`) are the only
  place an ephemeral value may go in a resource.

```hcl
resource "aws_ssm_parameter" "token" {
  name             = "/app/token"
  type             = "SecureString"
  value_wo         = var.token
  value_wo_version = 1
}
```

Change `value_wo_version` to make Terraform send a new value.

This applies to Floci (`tflings aws up`).
