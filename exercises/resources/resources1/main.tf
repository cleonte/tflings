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

# TODO: Terraform does not know the argument that names the bucket.
resource "aws_s3_bucket" "logs" {
  bucket_name   = "tflings-logs"
  force_destroy = true
}
