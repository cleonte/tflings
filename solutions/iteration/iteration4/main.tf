variable "enabled" {
  type    = bool
  default = true
}

resource "terraform_data" "feature" {
  count = var.enabled ? 1 : 0
  input = "on"
}
