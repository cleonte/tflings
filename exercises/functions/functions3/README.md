# functions3: cidrsubnet

`cidrsubnet(prefix, newbits, netnum)` cuts a network into smaller ones.
`newbits` is how many bits to add to the prefix length (`/16` + 8 = `/24`),
and `netnum` picks which of the resulting networks, counting from 0.

```hcl
cidrsubnet("10.0.0.0/16", 8, 0)   # "10.0.0.0/24"
cidrsubnet("10.0.0.0/16", 8, 1)   # "10.0.1.0/24"
```
