# I AM NOT DONE

variable "servers" {
  type    = list(string)
  default = ["web", "api", "db"]
}

# TODO: use for_each (over a set) instead of count, so that the instances are
# named after the servers and not numbered.
resource "terraform_data" "server" {
  count = length(var.servers)
  input = var.servers[count.index]
}
