resource "terraform_data" "app" {
  input = "app"
}

output "api_key" {
  value     = "ak-90-zz"
  sensitive = true
}
