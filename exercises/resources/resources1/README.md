# resources1: a resource and its arguments

A `resource` block describes one thing to create: its type (`aws_s3_bucket`), a
local name (`logs`) and arguments. Each resource type accepts only the
arguments its provider defines, and Terraform tells you when you use another.

This runs the real AWS provider against Floci, a local AWS that tflings starts
in Docker (`tflings aws up`). Nothing is created: this exercise only plans.

Fix the resource so the plan creates a bucket named `tflings-logs`.
