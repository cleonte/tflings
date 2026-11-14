# hcl1: reading an error

Terraform configuration is written in HCL. When a file has a syntax error,
Terraform stops before it plans anything and tells you the file, the line and
what it expected.

Run `tflings run hcl1`, read the error, find the line, and fix the file. The
output called `greeting` should be `Hello, tflings`.

This exercise uses only built-in things (`terraform_data`, `locals`, outputs):
no cloud, no provider download.
