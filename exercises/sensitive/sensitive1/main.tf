# I AM NOT DONE

# TODO: this API key must never be printed in a plan or in the logs.
# Mark the variable so that Terraform hides its value.
variable "api_key" {
  type    = string
  default = "abc123-not-a-real-key"
}

resource "terraform_data" "client" {
  input = var.api_key
}
