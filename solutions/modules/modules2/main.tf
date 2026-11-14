variable "name" {
  type    = string
  default = "orders"
}

module "app" {
  source = "./modules/app"
  name   = var.name
}

output "app_name" {
  value = module.app.name
}
