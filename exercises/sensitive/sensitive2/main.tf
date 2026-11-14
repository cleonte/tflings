# I AM NOT DONE

variable "db_password" {
  type      = string
  sensitive = true
  default   = "hunter2-not-a-real-password"
}

# TODO: Terraform refuses this output: it contains a sensitive value.
# Read the error, then fix the output.
output "connection_string" {
  value = "postgres://admin:${var.db_password}@db.internal:5432/app"
}
