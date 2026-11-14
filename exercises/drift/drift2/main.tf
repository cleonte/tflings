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

resource "aws_ssm_parameter" "motd" {
  name  = "/tflings/motd"
  type  = "String"
  value = "Welcome to tflings"
}
