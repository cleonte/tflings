locals {
  name = "tflings"
}

resource "terraform_data" "greeting" {
  input = "Hello, ${local.name}"
}

output "greeting" {
  value = terraform_data.greeting.input
}
