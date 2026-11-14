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

resource "aws_secretsmanager_secret_version" "db" {
  secret_id                = aws_secretsmanager_secret.db.id
  secret_string_wo         = var.db_password
  secret_string_wo_version = 1
}
