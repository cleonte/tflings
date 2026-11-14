variable "db_password" {
  type      = string
  sensitive = true
  default   = "hunter2-not-a-real-password"
}

output "connection_string" {
  value     = "postgres://admin:${var.db_password}@db.internal:5432/app"
  sensitive = true
}
