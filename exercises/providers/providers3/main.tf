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

provider "aws" {
  alias  = "west"
  region = "us-west-2"
}

# TODO: the backup bucket has to be built in us-west-2. A module uses the
# default provider configuration unless the module block passes another one.
module "backup" {
  source = "./modules/bucket"
  name   = "tflings-backup-copy"
}

module "main" {
  source = "./modules/bucket"
  name   = "tflings-main-copy"
}
