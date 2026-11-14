# I AM NOT DONE

variable "size" {
  type    = string
  default = "small"
}

# TODO: someone else resizes this server by hand, and that is fine.
# Terraform should stop trying to change the size back.
resource "terraform_data" "server" {
  input = var.size
}
