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

provider "aws" {
  alias  = "west"
  region = "us-west-2"
}

resource "aws_s3_bucket" "backup" {
  bucket        = "tflings-backup"
  force_destroy = true
  provider      = aws.west
}

resource "aws_s3_bucket" "main" {
  bucket        = "tflings-main"
  force_destroy = true
}
