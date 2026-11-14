# iteration3: dynamic blocks

Some arguments are nested blocks that you may repeat, like `ingress` in a
security group. A `dynamic` block generates one such block for every element of
a collection. Make the security group open exactly the ports in the list, with
one `ingress` rule each. tflings plans again with another list.

This plans against Floci but creates nothing.
