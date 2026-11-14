# I AM NOT DONE

variable "apps" {
  type    = list(string)
  default = ["web", "api"]
}

# TODO: one copy of the module for every app, each one named after its app.
module "app" {
  source = "./modules/app"
  name   = "web"
}
