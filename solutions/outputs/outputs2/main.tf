variable "nodes" {
  type    = number
  default = 3
}

resource "terraform_data" "node" {
  count = var.nodes
  input = "node-${count.index}"
}

output "node_names" {
  value = terraform_data.node[*].input
}
