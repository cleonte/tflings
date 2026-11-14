variable "users" {
  type    = list(string)
  default = ["alice", "bob"]
}

resource "terraform_data" "user" {
  for_each = toset(var.users)
  input    = "user-${each.key}"
}
