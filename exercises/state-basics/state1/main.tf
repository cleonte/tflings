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

resource "aws_ssm_parameter" "keep" {
  name  = "/tflings/app/name"
  type  = "String"
  value = "orders"
}

resource "aws_ssm_parameter" "legacy" {
  name  = "/tflings/legacy/flag"
  type  = "String"
  value = "on"
}
