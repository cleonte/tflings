# conditions2: preconditions

A `precondition` inside a resource's `lifecycle` block is checked before
Terraform plans that resource. Use it for assumptions about the inputs or about
other resources that a variable `validation` cannot express.

```hcl
resource "terraform_data" "app" {
  lifecycle {
    precondition {
      condition     = contains(["dev", "prod"], var.environment)
      error_message = "The environment must be dev or prod."
    }
  }
}
```

The failure message from Terraform says `Resource precondition failed`.
