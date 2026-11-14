# I AM NOT DONE

variable "overrides" {
  type    = map(string)
  default = {}
}

locals {
  defaults = {
    size   = "small"
    region = "eu"
  }
  # TODO: the defaults, with whatever the overrides say instead.
  settings = local.defaults
}

output "settings" {
  value = local.settings
}
