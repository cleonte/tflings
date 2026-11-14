# sensitive2: sensitive outputs

A value derived from a sensitive value is sensitive too. Terraform refuses to
print it in an output unless you say so, to stop a secret from leaking by
accident:

```hcl
output "connection_string" {
  value     = "postgres://admin:${var.db_password}@db"
  sensitive = true
}
```

A sensitive output is hidden in the CLI, but `terraform output -raw NAME` and
`-json` still print it, and it is stored in the state.
