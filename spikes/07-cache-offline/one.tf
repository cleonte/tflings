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

resource "aws_ssm_parameter" "greeting" {
  name  = "/spike07/greeting"
  type  = "String"
  value = "hello"
}
