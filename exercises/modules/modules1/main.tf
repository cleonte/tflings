# I AM NOT DONE

variable "name" {
  type    = string
  default = "orders"
}

# TODO: the module needs an input called name. Terraform tells you so: read the
# error, then pass the variable of this configuration to the module.
module "app" {
  source = "./modules/app"
}
