variable "size" {
  type    = string
  default = "small"
}

resource "terraform_data" "server" {
  input = var.size

  lifecycle {
    ignore_changes = [input]
  }
}
