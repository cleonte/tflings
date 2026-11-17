variable "name" {
  type = string
}

output "message" {
  value = "Hello, ${var.name}"
}
