variable "servers" {
  type = map(string)
  default = {
    web = "10.0.0.1"
    db  = "10.0.0.2"
  }
}

output "addresses" {
  value = { for name, ip in var.servers : upper(name) => ip }
}
