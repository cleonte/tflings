# providers1: more than one provider configuration

A `provider` block configures a provider. To use the same provider twice, for
example in two regions, give the second block an `alias` and select it in a
resource with `provider = aws.<alias>`. A resource that does not say anything
uses the default (not aliased) configuration.

```hcl
provider "aws" {
  alias  = "west"
  region = "us-west-2"
}

resource "aws_s3_bucket" "backup" {
  provider = aws.west
  # ...
}
```

This plans against Floci (`tflings aws up`); nothing is created.
