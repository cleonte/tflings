# I AM NOT DONE

resource "terraform_data" "web" {
  input = "web-server"
}

output "web_name" {
  # TODO: Terraform says this resource has no such attribute.
  value = terraform_data.web.name
}
