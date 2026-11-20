terraform {
  required_version = ">= 1.12"
}

resource "terraform_data" "this" {
  input = "app"
}
