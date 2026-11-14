# conditions3: preconditions on outputs

Outputs take `precondition` blocks too, written directly in the `output` block.
They guard what a module promises to its callers.

```hcl
output "replicas" {
  value = var.replicas

  precondition {
    condition     = var.replicas % 2 == 1
    error_message = "The number of replicas must be odd."
  }
}
```
