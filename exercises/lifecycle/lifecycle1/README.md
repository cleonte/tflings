# lifecycle1: create_before_destroy

Some changes cannot be made in place: Terraform replaces the resource. By
default it destroys the old one first and then creates the new one. With
`create_before_destroy = true` in a `lifecycle` block it is the other way
round, which is what you want when something depends on the resource.

tflings builds your configuration with release `v1`, then plans it with `v2`,
and looks at the order of the replacement.
