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

# TODO: the backup bucket has to live in us-west-2, not in the default region.
# Add a second configuration of the aws provider, with an alias, and select it.
resource "aws_s3_bucket" "backup" {
  bucket        = "tflings-backup"
  force_destroy = true
  provider      = aws.west
}

resource "aws_s3_bucket" "main" {
  bucket        = "tflings-main"
  force_destroy = true
}
