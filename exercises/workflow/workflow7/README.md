# workflow7: init installs the modules

`terraform init` prepares a working directory. Among other things it finds the
modules a configuration calls and installs them (a local module is only linked,
a registry or Git module is downloaded), and it downloads the providers. Until
`init` has worked, no other command does: `plan` and `apply` refuse to start.

Here `init` stops, because the `module` block points at a folder that is not
there. Run `tflings run workflow7`, read what Terraform says about the module,
and fix the `source`. A change to a module's `source` is one of the reasons to run
`terraform init` again.

The module itself, in `modules/greeting/`, is fine.

Related tutorial: https://developer.hashicorp.com/terraform/tutorials/cli/init
