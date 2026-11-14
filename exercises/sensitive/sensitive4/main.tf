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

variable "db_password" {
  type      = string
  ephemeral = true
  default   = "hunter2-not-a-real-password"
}

resource "aws_secretsmanager_secret" "db" {
  name                    = "tflings-db-password"
  recovery_window_in_days = 0
}

# TODO: the password must reach Secrets Manager without ever being written to
# the state. Use the write-only argument of the secret version.
resource "aws_secretsmanager_secret_version" "db" {
  secret_id     = aws_secretsmanager_secret.db.id
  secret_string = var.db_password
}
