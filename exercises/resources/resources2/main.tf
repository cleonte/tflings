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

# TODO: this bucket should keep every version of its objects.
resource "aws_s3_bucket" "backups" {
  bucket        = "tflings-backups"
  force_destroy = true
}
