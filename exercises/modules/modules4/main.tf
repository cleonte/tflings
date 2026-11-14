# I AM NOT DONE

variable "environment" {
  type    = string
  default = "prod"
}

# TODO: the module cannot see the variables of this configuration. Give it an
# input of its own (see modules/app/main.tf) and pass the value here.
module "app" {
  source = "./modules/app"
}
