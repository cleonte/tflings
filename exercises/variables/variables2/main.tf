# I AM NOT DONE

variable "instance_count" {
  type = number
  # TODO: the default does not match the type.
  default = "three"
}

output "total_cost" {
  value = var.instance_count * 10
}
