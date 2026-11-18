variable "tier" {
  type    = string
  default = "small"
}

resource "terraform_data" "app" {
  input = var.teir
}

output "app_tier" {
  value = terraform_data.application.input
}
