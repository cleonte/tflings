variable "name" {
  type    = string
  default = "tflings"
}

output "name" {
  value = var.name
}
