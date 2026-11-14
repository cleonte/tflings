# I AM NOT DONE

variable "environment" {
  type    = string
  default = "prod"
}

locals {
  # TODO: production needs 3 replicas, every other environment needs 1.
  replicas = 1
}

output "replicas" {
  value = local.replicas
}
