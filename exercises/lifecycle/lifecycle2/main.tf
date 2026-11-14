# I AM NOT DONE

variable "name" {
  type    = string
  default = "orders"
}

# TODO: this database must never be destroyed by a plan, not even by accident.
resource "terraform_data" "database" {
  triggers_replace = var.name
  input            = var.name
}
