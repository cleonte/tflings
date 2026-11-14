# I AM NOT DONE

variable "users" {
  type    = list(string)
  default = ["alice", "bob"]
}

# TODO: one resource for every user, named by the user, holding "user-" and the name.
# for_each does not accept a list as it is.
resource "terraform_data" "user" {
  for_each = var.users
  input    = "user-${each.key}"
}
