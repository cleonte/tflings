# references1: refer to another resource

A resource can use an attribute of another resource: `aws_s3_bucket.assets.id`
is the id of the bucket that Terraform manages. Written like that, Terraform
knows the bucket has to exist first, and the two can never drift apart. Writing
the name out by hand loses both.

This exercise really creates things, in a fake AWS account of its own in Floci
(`tflings aws up`). Run it, read what Terraform says when the apply fails, and
make the object go into the bucket that this configuration creates, by
reference.
