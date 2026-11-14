# sensitive4: write-only secrets in Secrets Manager

Same idea as `sensitive3`, with another service. `aws_secretsmanager_secret_version`
has a write-only argument, `secret_string_wo`, and its version,
`secret_string_wo_version`:

```hcl
resource "aws_secretsmanager_secret_version" "db" {
  secret_id                = aws_secretsmanager_secret.db.id
  secret_string_wo         = var.db_password
  secret_string_wo_version = 1
}
```

Terraform never stores the value, so it cannot see a change: to rotate the
secret, raise the version number.

This applies to Floci (`tflings aws up`).
