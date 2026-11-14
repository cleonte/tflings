# conditions4: check blocks

A `check` block holds one or more `assert` blocks. A failing assertion prints a
**warning**; it never stops a plan or an apply. Use it for rules that are worth
reporting but not worth blocking on.

```hcl
check "retention" {
  assert {
    condition     = var.retention_days >= 30
    error_message = "Retention is under 30 days."
  }
}
```
