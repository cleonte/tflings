variable "replicas" {
  type    = number
  default = 3
}

output "replicas" {
  value = var.replicas

  precondition {
    condition     = var.replicas % 2 == 1
    error_message = "The number of replicas must be odd."
  }
}
