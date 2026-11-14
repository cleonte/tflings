variable "name" {
  type    = string
  default = "orders"
}

resource "terraform_data" "database" {
  triggers_replace = var.name
  input            = var.name

  lifecycle {
    prevent_destroy = true
  }
}
