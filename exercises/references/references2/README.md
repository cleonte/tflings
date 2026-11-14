# references2: ids come from other resources

Most resources need the id of another one: a subnet needs the VPC it lives in.
You do not know that id before the VPC exists, which is exactly why Terraform
lets you refer to it: `aws_vpc.main.id`.

Run the exercise, read the error, and put the subnet in the VPC that this
configuration creates.
