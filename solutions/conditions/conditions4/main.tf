variable "retention_days" {
  type    = number
  default = 7
}

resource "terraform_data" "logs" {
  input = var.retention_days
}

check "retention" {
  assert {
    condition     = var.retention_days >= 30
    error_message = "Retention is under 30 days."
  }
}
