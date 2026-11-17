variable "tier" {
  type    = string
  default = "small"
}

resource "terraform_data" "app" {
  input = var.tier
}

output "app_tier" {
  value = terraform_data.app.input
}
