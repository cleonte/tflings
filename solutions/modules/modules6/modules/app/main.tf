resource "terraform_data" "this" {
  input = "app"
}

# The app module needs the db module, which sits next to it in ./modules.
module "db" {
  source = "../db"
}
