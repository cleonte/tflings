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

resource "aws_s3_bucket" "assets" {
  bucket        = "tflings-assets"
  force_destroy = true
}

# TODO: this object is stored in a bucket that does not exist.
resource "aws_s3_object" "logo" {
  bucket  = "tflings-asset"
  key     = "logo.txt"
  content = "tflings"
}
