resource "terraform_data" "server" {
  input = "web-1"
}

moved {
  from = terraform_data.web
  to   = terraform_data.server
}
