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

data "aws_vpc" "default" {
  default = true
}

resource "aws_security_group" "web" {
  name        = "tflings-web"
  description = "Web servers"
  vpc_id      = data.aws_vpc.default.id
}
