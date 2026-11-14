resource "terraform_data" "database" {
  input = "orders-db"
}

resource "terraform_data" "app" {
  input = "orders-app for ${terraform_data.database.input}"
}
