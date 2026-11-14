variable "port" {
  type    = number
  default = 8080

  validation {
    condition     = var.port >= 1 && var.port <= 65535
    error_message = "The port must be between 1 and 65535."
  }
}

output "port" {
  value = var.port
}
