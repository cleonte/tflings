# I AM NOT DONE

variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}

# TODO: the second /24 network inside the VPC range: 10.0.1.0/24 for the default.
output "subnet_b" {
  value = var.vpc_cidr
}
