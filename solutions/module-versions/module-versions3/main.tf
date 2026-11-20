terraform {
  required_version = ">= 1.12"
}

module "app" {
  source = "./modules/app"
}
