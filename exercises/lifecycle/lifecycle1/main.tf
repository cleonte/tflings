# I AM NOT DONE

variable "release" {
  type    = string
  default = "v1"
}

# TODO: when the release changes this server is replaced. The new one has to
# exist before the old one is removed, so that nothing is ever without a server.
resource "terraform_data" "server" {
  triggers_replace = var.release
  input            = "server-${var.release}"
}
