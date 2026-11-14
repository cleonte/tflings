# I AM NOT DONE

variable "name" {
  type    = string
  default = "orders"
}

module "app" {
  source = "./modules/app"
  name   = var.name
}

# TODO: Terraform refuses this: the resources inside a module are private. The
# module must export the value as an output (see modules/app/outputs.tf), and
# this output must read the module's output.
output "app_name" {
  value = module.app.terraform_data.this.input
}
