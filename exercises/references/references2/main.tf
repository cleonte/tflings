# I AM NOT DONE

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

# TODO: this subnet is created in a VPC that does not exist.
resource "aws_subnet" "app" {
  vpc_id     = "vpc-0123456789abcdef0"
  cidr_block = "10.0.1.0/24"
}
