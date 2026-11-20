# I AM NOT DONE

terraform {
  required_version = ">= 1.12"
}

# TODO: init refuses this module. Read the error: it names the file of the
# module where the Terraform version constraint has to change.
module "app" {
  source = "./modules/app"
}
