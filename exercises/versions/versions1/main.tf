# I AM NOT DONE

terraform {
  # TODO: this pins one old release. Allow Terraform 1.12 and every newer 1.x
  # release, but not 2.0: use the pessimistic operator ~>.
  required_version = "= 1.5.0"
}

resource "terraform_data" "app" {
  input = "hello"
}
