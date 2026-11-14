# I AM NOT DONE

# TODO: only real TCP ports are acceptable: 1 to 65535.
# Make Terraform reject any other value before it plans anything.
variable "port" {
  type    = number
  default = 8080
}

output "port" {
  value = var.port
}
