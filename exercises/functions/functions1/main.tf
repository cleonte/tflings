# I AM NOT DONE

variable "name" {
  type    = string
  default = "tflings"
}

output "shout" {
  # TODO: Terraform does not know this function.
  value = uppercase(var.name)
}
