# data-sources2: look it up

A `data` block reads an existing object instead of creating one, and the rest of
the configuration uses its attributes like any other reference:

```hcl
data "aws_vpc" "default" {
  default = true
}

# ... vpc_id = data.aws_vpc.default.id
```

Never type an id that Terraform can look up: it is different in every account.
This applies to Floci (`tflings aws up`).
