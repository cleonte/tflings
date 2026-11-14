# iteration4: count as an on/off switch

`count = 0` builds nothing, `count = 1` builds one. With a conditional it makes
a resource optional:

```hcl
count = var.enabled ? 1 : 0
```

The resource then has an index: `terraform_data.feature[0]`.
