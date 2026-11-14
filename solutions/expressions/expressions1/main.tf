variable "environment" {
  type    = string
  default = "prod"
}

locals {
  replicas = var.environment == "prod" ? 3 : 1
}

output "replicas" {
  value = local.replicas
}
