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

resource "aws_vpc" "main" {
  cidr_block = "10.1.0.0/16"

  # TODO: every VPC here needs the tags Environment = "dev" and Owner = "tflings".
  tags = {
    Name = "main"
  }
}
