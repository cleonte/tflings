variable "settings" {
  type = map(string)
  default = {
    name = "app"
  }
}

output "timeout" {
  value = lookup(var.settings, "timeout", "30")
}
