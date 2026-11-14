# I AM NOT DONE

variable "environment" {
  type    = string
  default = "prod"
}

# TODO: this resource may only be created for "dev", "staging" or "prod".
# Add a precondition that stops the plan for any other environment.
resource "terraform_data" "app" {
  input = var.environment
}
