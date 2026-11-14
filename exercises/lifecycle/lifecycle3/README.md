# lifecycle3: ignore_changes

When something else also changes an attribute (an autoscaler, a person in the
console), Terraform would keep changing it back. `ignore_changes` lists the
attributes to leave alone after the resource is created.

tflings builds your configuration with size `small`, then plans it with
`large`, and expects Terraform to see nothing to do.
