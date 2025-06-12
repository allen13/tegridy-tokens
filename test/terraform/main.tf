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

# Use the Lambda module for testing
module "tegridy_tokens" {
  source = "../../terraform/tegridy"

  project_name = "tegridy-test"
  environment  = "test"
  aws_region   = var.aws_region

  # Lambda configuration
  lambda_timeout     = 30
  lambda_memory      = 512
  lambda_source_path = "${path.module}/../../cmd/lambda"

  # DynamoDB configuration
  dynamodb_billing_mode = "PAY_PER_REQUEST"

  # KMS configuration
  kms_deletion_window = 7
  kms_enable_rotation = false # Disabled for testing

  # Logging configuration
  log_retention_days = 1

  # X-Ray tracing
  enable_xray_tracing = false

  tags = {
    Environment = "test"
    ManagedBy   = "terratest"
    Purpose     = "integration-testing"
  }
}
