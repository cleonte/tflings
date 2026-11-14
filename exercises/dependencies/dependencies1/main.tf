# I AM NOT DONE

resource "terraform_data" "database" {
  input = "database"
}

# TODO: the app must only be created after the database, but nothing in it
# refers to the database, so Terraform does not know about the order.
resource "terraform_data" "app" {
  input = "app"
}
