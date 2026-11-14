# I AM NOT DONE

# TODO: Terraform cannot plan this because a value is missing, and tflings
# does not answer questions. Make the variable work without being asked.
variable "environment" {
  type        = string
  description = "Which environment this is."
}

output "name" {
  value = "app-${var.environment}"
}
