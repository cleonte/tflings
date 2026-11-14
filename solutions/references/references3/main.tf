resource "terraform_data" "web" {
  input = "web-server"
}

output "web_name" {
  value = terraform_data.web.input
}
