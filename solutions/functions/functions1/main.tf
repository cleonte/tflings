variable "name" {
  type    = string
  default = "tflings"
}

output "shout" {
  value = upper(var.name)
}
