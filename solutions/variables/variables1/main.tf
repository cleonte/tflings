variable "environment" {
  type        = string
  description = "Which environment this is."
  default     = "dev"
}

output "name" {
  value = "app-${var.environment}"
}
