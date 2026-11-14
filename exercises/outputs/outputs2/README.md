# outputs2: splat expressions

A resource with `count` is a list of instances, so `terraform_data.node.input`
has no meaning. The splat expression reads one attribute of every instance:

```hcl
terraform_data.node[*].input
```

It gives a list, in the order of the instances.
