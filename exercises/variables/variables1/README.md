# variables1: a variable needs a value

An input variable without a `default` has no value until you give it one, with
`-var`, a `.tfvars` file or an environment variable. Terraform normally asks for
it; when it cannot ask, planning fails.

Here the output `name` should be `app-dev`. Make the configuration plan on its
own, and keep the variable as it is declared.
