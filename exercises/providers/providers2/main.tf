# I AM NOT DONE

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      # TODO: this constraint excludes every version that exists today.
      # Allow any 6.x release (6.0 or newer, but not 7.0) with the pessimistic operator.
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_s3_bucket" "logs" {
  bucket        = "tflings-logs"
  force_destroy = true
}
