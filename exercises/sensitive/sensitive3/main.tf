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

# An ephemeral variable exists only while Terraform runs: it is never written
# to the plan or to the state.
variable "token" {
  type      = string
  ephemeral = true
  default   = "s3cr3t-not-a-real-token"
}

# TODO: an ephemeral value cannot go into a normal argument, which is saved in
# the state. Use the write-only argument of the parameter instead, and a
# version number so that Terraform knows when to send a new value.
resource "aws_ssm_parameter" "token" {
  name  = "/tflings/token"
  type  = "SecureString"
  value = var.token
}
