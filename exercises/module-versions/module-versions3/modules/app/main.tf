terraform {
  # The author of this module wrote it for Terraform 1.0 and pinned exactly
  # that. Allow 1.12 and every newer release instead, with an operator that
  # sets a minimum.
  required_version = "= 1.0.0"
}

resource "terraform_data" "this" {
  input = "app"
}
