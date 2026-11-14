locals {
  servers = ["web", "api", "db"]
}

resource "terraform_data" "server" {
  for_each = toset(local.servers)
  input    = each.key
}

output "server_count" {
  value = length(terraform_data.server)
}
