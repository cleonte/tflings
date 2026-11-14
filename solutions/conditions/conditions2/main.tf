variable "environment" {
  type    = string
  default = "prod"
}

resource "terraform_data" "app" {
  input = var.environment

  lifecycle {
    precondition {
      condition     = contains(["dev", "staging", "prod"], var.environment)
      error_message = "The environment must be dev, staging or prod."
    }
  }
}
