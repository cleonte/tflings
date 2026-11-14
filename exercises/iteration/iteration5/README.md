# iteration5: for_each over a list

`for_each` wants a **map** or a **set of strings**, because every instance needs
a unique key. A list may repeat values and has an order, so Terraform refuses
it. `toset()` converts it:

```hcl
for_each = toset(var.users)
```

For a set, `each.key` and `each.value` are the same string.
