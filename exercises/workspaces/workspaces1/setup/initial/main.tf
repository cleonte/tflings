resource "terraform_data" "app" {
  input = "app-${terraform.workspace}"
}
