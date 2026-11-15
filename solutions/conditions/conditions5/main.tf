variable "bucket_name" {
  type    = string
  default = "tflings-logs"

  validation {
    condition     = can(regex("^[a-z0-9-]{3,20}$", var.bucket_name))
    error_message = "Use 3 to 20 characters: lower case letters, digits and dashes."
  }
}

output "name" {
  value = var.bucket_name
}
