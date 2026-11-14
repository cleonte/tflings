variable "allowed_ports" {
  type    = list(number)
  default = [80, 443]
}

output "port_count" {
  value = length(var.allowed_ports)
}
