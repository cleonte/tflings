# I AM NOT DONE

# TODO: Terraform cannot even read this file. Find the syntax error.
locals {
  name = "tflings"
}

resource "terraform_data" "greeting" {
  input = "Hello, ${local.name}
}

output "greeting" {
  value = terraform_data.greeting.input
}
