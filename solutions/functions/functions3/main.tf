variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}

output "subnet_b" {
  value = cidrsubnet(var.vpc_cidr, 8, 1)
}
