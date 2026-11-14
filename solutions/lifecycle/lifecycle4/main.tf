variable "config_version" {
  type    = string
  default = "1"
}

resource "terraform_data" "config" {
  input = var.config_version
}

resource "terraform_data" "app" {
  input = "app"

  lifecycle {
    replace_triggered_by = [terraform_data.config]
  }
}
