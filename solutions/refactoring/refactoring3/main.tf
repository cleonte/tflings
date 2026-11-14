module "web" {
  source = "./modules/web"
}

moved {
  from = terraform_data.web
  to   = module.web.terraform_data.this
}
