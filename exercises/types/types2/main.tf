# I AM NOT DONE

variable "app" {
  # TODO: a map(string) cannot hold a list. Terraform names the problem in its
  # error. Use a type that gives each attribute its own type.
  type = map(string)
  default = {
    name  = "web"
    ports = [80, 443]
  }
}

output "port_count" {
  value = length(var.app.ports)
}

output "label" {
  value = "${var.app.name}:${var.app.ports[0]}"
}
