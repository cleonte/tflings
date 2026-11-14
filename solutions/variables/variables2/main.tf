variable "instance_count" {
  type    = number
  default = 3
}

output "total_cost" {
  value = var.instance_count * 10
}
