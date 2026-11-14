# iteration6: for expressions

A `for` expression builds a new list or map from another collection:

```hcl
[for ip in var.ips : upper(ip)]                 # a list
{ for name, ip in var.servers : upper(name) => ip }   # a map, braces and =>
```

Square brackets give a list, braces give a map.
