# conditions5: a validation that does not blow up

A `validation` block's `condition` must be a **bool**. `regex()` does not return a bool: it
returns the matching text, and it **fails with an error** when there is no match.

`can(expression)` returns `true` if the expression works and `false` if it raises an
error, so `can(regex(...))` is the usual way to ask "does this text match?".

```hcl
condition = can(regex("^[a-z]+$", var.name))
```

This validation uses `regex` directly. Fix it, so that good names plan and bad names are
refused with the `error_message`.
