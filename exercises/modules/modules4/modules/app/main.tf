# TODO: this variable is not declared in the module, so Terraform cannot find it.
resource "terraform_data" "this" {
  input = "app-${var.environment}"
}
