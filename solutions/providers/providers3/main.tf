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

module "backup" {
  source = "./modules/bucket"
  name   = "tflings-backup-copy"

  providers = {
    aws = aws.west
  }
}

module "main" {
  source = "./modules/bucket"
  name   = "tflings-main-copy"
}
