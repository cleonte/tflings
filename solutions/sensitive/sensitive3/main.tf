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

variable "token" {
  type      = string
  ephemeral = true
  default   = "s3cr3t-not-a-real-token"
}

resource "aws_ssm_parameter" "token" {
  name             = "/tflings/token"
  type             = "SecureString"
  value_wo         = var.token
  value_wo_version = 1
}
