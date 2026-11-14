# I AM NOT DONE

variable "nodes" {
  type    = number
  default = 3
}

resource "terraform_data" "node" {
  count = var.nodes
  input = "node-${count.index}"
}

# TODO: list the input of every node. Terraform does not accept this as it is:
# a resource with count has many instances.
output "node_names" {
  value = terraform_data.node.input
}
