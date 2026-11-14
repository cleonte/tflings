# data-sources1: reading what already exists

A `data` block reads something that exists, without creating it. The AWS
provider can tell you who you are: the data source `aws_caller_identity` has
the account id. Use it for the output `account_id` instead of the number that
was typed in.

This runs against Floci with an account of its own.
