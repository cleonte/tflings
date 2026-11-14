resource "terraform_data" "database" {
  input = "database"
}

resource "terraform_data" "app" {
  input      = "app"
  depends_on = [terraform_data.database]
}
