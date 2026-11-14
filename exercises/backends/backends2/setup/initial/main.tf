terraform {
  backend "s3" {
    bucket       = "tflings-key-state"
    key          = "app/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
  }
}

resource "terraform_data" "app" {
  input = "v1"
}
