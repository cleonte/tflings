variable "name" {
  type    = string
  default = "tflings"
}

module "greeting" {
  source = "./modules/greeting"
  name   = var.name
}

output "greeting" {
  value = module.greeting.message
}
