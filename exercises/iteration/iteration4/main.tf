# I AM NOT DONE

variable "enabled" {
  type    = bool
  default = true
}

# TODO: the feature should only be built when var.enabled is true.
resource "terraform_data" "feature" {
  input = "on"
}
