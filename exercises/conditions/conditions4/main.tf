# I AM NOT DONE

variable "retention_days" {
  type    = number
  default = 7
}

resource "terraform_data" "logs" {
  input = var.retention_days
}

# TODO: retention under 30 days is allowed, but worth a warning.
# Add a check block named "retention" with an assert that does not stop the plan.
