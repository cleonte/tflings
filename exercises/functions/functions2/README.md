# functions2: merge

`merge()` combines maps into one. When two maps have the same key, the **last**
one wins, which makes it the usual way to apply overrides to defaults:

```hcl
merge(local.defaults, var.overrides)
```
