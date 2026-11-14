# I AM NOT DONE

variable "settings" {
  type = map(string)
  default = {
    name = "app"
  }
}

# TODO: the map may have no "timeout" key. Use "30" then, instead of failing.
output "timeout" {
  value = var.settings["timeout"]
}
