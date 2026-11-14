# resources2: settings that are resources of their own

In the AWS provider many settings of a bucket are not arguments of
`aws_s3_bucket` but separate resources that point at it. Versioning is one:
there is a resource for it.

This exercise creates things in your fake AWS account. Get the bucket
`tflings-backups` to keep every version of its objects, then run it: tflings
asks Floci whether versioning really is enabled.
