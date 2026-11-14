variable "names" {
  type    = list(string)
  default = ["web", "api", "db"]
}

resource "terraform_data" "server" {
  count = length(var.names)
  input = var.names[count.index]
}
