terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  backend "s3" {}
}

provider "aws" {
  region = "us-east-1"
}

variable "id" {
  type    = string
  default = "x"
}

variable "hold" {
  type    = number
  default = 0
}

resource "aws_ssm_parameter" "greeting" {
  name  = "/spike03/${var.id}"
  type  = "String"
  value = "hello"
}

# Holds the lock for `hold` seconds so the run can be killed mid-apply.
resource "terraform_data" "hold" {
  depends_on = [aws_ssm_parameter.greeting]
  provisioner "local-exec" {
    command = "sleep ${var.hold}"
  }
}
