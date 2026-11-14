# expressions2: a value that may be missing

Reading a key that is not in a map is an error. Two ways to give a fallback:

```hcl
lookup(var.settings, "timeout", "30")
try(var.settings["timeout"], "30")
```

`try()` is more general: it returns the first expression that does not fail.
