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

# TODO: this is the account the configuration runs in, but someone typed it in.
# Ask the provider instead, with a data source.
output "account_id" {
  value = "123456789012"
}
