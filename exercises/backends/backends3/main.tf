terraform {
  backend "local" {
    path = "state/app.tfstate"
  }
}

resource "terraform_data" "app" {
  input = "app"
}
