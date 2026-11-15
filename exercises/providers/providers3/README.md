# providers3: pass a provider to a module

A module uses the **default** configuration of each provider it needs (the `provider`
block without an `alias`). To make one module call use another configuration, for
example another region, pass it in the `module` block:

```hcl
module "backup" {
  source = "./modules/bucket"

  providers = {
    aws = aws.west     # inside the module, "aws" is the root's aws.west
  }
}
```

Both buckets of this exercise are built by the same module. The backup copy has to be
in `us-west-2`; the main copy stays in the default region. This plans against Floci
(`tflings aws up`); nothing is created.
