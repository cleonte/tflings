# I AM NOT DONE

locals {
  servers = ["web", "api", "db"]
}

resource "terraform_data" "server" {
  for_each = toset(local.servers)
  input    = each.key
}

# TODO: add an output called server_count with the number of servers.
