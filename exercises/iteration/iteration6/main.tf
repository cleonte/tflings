# I AM NOT DONE

variable "servers" {
  type = map(string)
  default = {
    web = "10.0.0.1"
    db  = "10.0.0.2"
  }
}

# TODO: the same map, but with every key in capital letters.
output "addresses" {
  value = var.servers
}
