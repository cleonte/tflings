variable "environment" {
  type    = string
  default = "prod"
}

module "app" {
  source      = "./modules/app"
  environment = var.environment
}
