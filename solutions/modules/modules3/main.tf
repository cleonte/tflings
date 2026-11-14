variable "apps" {
  type    = list(string)
  default = ["web", "api"]
}

module "app" {
  source   = "./modules/app"
  for_each = toset(var.apps)
  name     = each.key
}
