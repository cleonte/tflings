# I AM NOT DONE

# TODO: this file works, but it is not formatted the way terraform fmt writes it.
variable   "name"   {
type=string
  default    = "tflings"
}

output "name" {
    value = var.name
}
