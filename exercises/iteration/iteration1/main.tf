# I AM NOT DONE

variable "names" {
  type    = list(string)
  default = ["web", "api", "db"]
}

# TODO: there should be one of these for every name in the list.
resource "terraform_data" "server" {
  input = var.names[0]
}
