variable "servers" {
  type    = list(string)
  default = ["web", "api", "db"]
}

resource "terraform_data" "server" {
  for_each = toset(var.servers)
  input    = each.key
}
