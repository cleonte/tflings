variable "ports" {
  type = map(number)
  default = {
    http  = 80
    https = 443
  }
}

resource "terraform_data" "port" {
  for_each = var.ports
  input    = each.value
}
