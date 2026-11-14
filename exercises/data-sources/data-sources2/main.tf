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

# TODO: the security group belongs in the default VPC of the account, and the id
# was typed in by hand (and is wrong). Look the VPC up with a data source.
resource "aws_security_group" "web" {
  name        = "tflings-web"
  description = "Web servers"
  vpc_id      = "vpc-0123456789abcdef0"
}
