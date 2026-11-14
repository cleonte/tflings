variable "environment" {
  type = string
}

resource "terraform_data" "this" {
  input = "app-${var.environment}"
}
