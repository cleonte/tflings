variable "overrides" {
  type    = map(string)
  default = {}
}

locals {
  defaults = {
    size   = "small"
    region = "eu"
  }
  settings = merge(local.defaults, var.overrides)
}

output "settings" {
  value = local.settings
}
