# I AM NOT DONE

variable "config_version" {
  type    = string
  default = "1"
}

resource "terraform_data" "config" {
  input = var.config_version
}

# TODO: the app only reads its configuration when it starts, so it must be
# replaced whenever the configuration changes.
resource "terraform_data" "app" {
  input = "app"
}
