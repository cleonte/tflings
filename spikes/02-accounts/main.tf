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

data "aws_caller_identity" "current" {}
data "aws_availability_zones" "available" {}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  tags       = { Name = "spike01" }
}

resource "aws_subnet" "a" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]
}

resource "aws_security_group" "web" {
  name        = "spike01-web"
  description = "spike01 web"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_s3_bucket" "data" {
  bucket = "spike02-same-name"
}

resource "aws_s3_bucket_versioning" "data" {
  bucket = aws_s3_bucket.data.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_object" "hello" {
  depends_on = [aws_s3_bucket_versioning.data]
  bucket  = aws_s3_bucket.data.id
  key     = "hello.txt"
  content = "hello"
}

resource "aws_iam_role" "app" {
  name = "spike01-app"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Action    = "sts:AssumeRole"
      Principal = { Service = "ec2.amazonaws.com" }
    }]
  })
}

resource "aws_iam_policy" "read" {
  name = "spike01-read"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["s3:GetObject"]
      Resource = "${aws_s3_bucket.data.arn}/*"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "read" {
  role       = aws_iam_role.app.name
  policy_arn = aws_iam_policy.read.arn
}

resource "aws_ssm_parameter" "greeting" {
  name  = "/spike01/greeting"
  type  = "String"
  value = "hello"
}

resource "aws_secretsmanager_secret" "db" {
  name                    = "spike01/db"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "db" {
  secret_id     = aws_secretsmanager_secret.db.id
  secret_string = "s3cr3t"
}

resource "aws_dynamodb_table" "items" {
  name         = "spike01-items"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }
}

output "account_id" {
  value = data.aws_caller_identity.current.account_id
}
