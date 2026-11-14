# I AM NOT DONE

resource "terraform_data" "database" {
  input = "orders-db"
}

# TODO: the app must be built after the database, and Terraform must find that
# out from the configuration itself, with a reference. Do not use depends_on.
resource "terraform_data" "app" {
  input = "orders-app"
}
