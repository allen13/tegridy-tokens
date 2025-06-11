terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Random suffix for unique resource names
resource "random_id" "suffix" {
  byte_length = 4
}

# KMS Key for tokenization
resource "aws_kms_key" "tokenization" {
  description             = "KMS key for tegridy-tokens integration test ${random_id.suffix.hex}"
  deletion_window_in_days = 7
  
  tags = {
    Name        = "tegridy-tokens-test-${random_id.suffix.hex}"
    Environment = "test"
    ManagedBy   = "terratest"
  }
}

resource "aws_kms_alias" "tokenization" {
  name          = "alias/tegridy-tokens-test-${random_id.suffix.hex}"
  target_key_id = aws_kms_key.tokenization.key_id
}

# DynamoDB table for storing tokens
resource "aws_dynamodb_table" "tokens" {
  name           = "tegridy-tokens-test-${random_id.suffix.hex}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "token_value"
  
  # Enable TTL on the ttl attribute
  ttl {
    attribute_name = "ttl"
    enabled        = true
  }
  
  attribute {
    name = "token_value"
    type = "S"
  }
  
  # Enable point-in-time recovery for production use
  point_in_time_recovery {
    enabled = true
  }
  
  # Enable server-side encryption
  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.tokenization.arn
  }
  
  tags = {
    Name        = "tegridy-tokens-test-${random_id.suffix.hex}"
    Environment = "test"
    ManagedBy   = "terratest"
  }
}

# IAM role for testing (optional, for Lambda deployment)
resource "aws_iam_role" "lambda_role" {
  name = "tegridy-tokens-lambda-test-${random_id.suffix.hex}"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
  
  tags = {
    Name        = "tegridy-tokens-lambda-test-${random_id.suffix.hex}"
    Environment = "test"
    ManagedBy   = "terratest"
  }
}

# IAM policy for Lambda to access DynamoDB and KMS
resource "aws_iam_role_policy" "lambda_policy" {
  name = "tegridy-tokens-lambda-policy-${random_id.suffix.hex}"
  role = aws_iam_role.lambda_role.id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:BatchWriteItem",
          "dynamodb:BatchGetItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = aws_dynamodb_table.tokens.arn
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:DescribeKey"
        ]
        Resource = aws_kms_key.tokenization.arn
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      }
    ]
  })
}