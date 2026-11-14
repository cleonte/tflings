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

variable "ports" {
  type    = list(number)
  default = [80, 443, 8080]
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_security_group" "web" {
  name   = "tflings-web"
  vpc_id = aws_vpc.main.id

  # TODO: one rule for every port in the list. This one is written out by hand.
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
