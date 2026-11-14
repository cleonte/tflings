variable "release" {
  type    = string
  default = "v1"
}

resource "terraform_data" "server" {
  triggers_replace = var.release
  input            = "server-${var.release}"

  lifecycle {
    create_before_destroy = true
  }
}
