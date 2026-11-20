variable "app" {
  type = object({
    name  = string
    ports = list(number)
  })
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
