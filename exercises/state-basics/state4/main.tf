resource "terraform_data" "web" {
  input = "web-1"
}

resource "terraform_data" "api" {
  input = "api-v1"
}

resource "terraform_data" "db" {
  input = "db-primary"
}
