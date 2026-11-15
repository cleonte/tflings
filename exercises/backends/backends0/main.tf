# I AM NOT DONE

terraform {
  # TODO: set where the local backend writes the state file.
  backend "local" {}
}

resource "terraform_data" "app" {
  input = "app"
}
