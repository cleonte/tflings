# I AM NOT DONE

variable "allowed_ports" {
  type = list(number)
  # TODO: this is not a list of numbers.
  default = "80,443"
}

output "port_count" {
  value = length(var.allowed_ports)
}
