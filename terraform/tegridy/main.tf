terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.4"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }
}

# Random suffix for unique resource naming
resource "random_id" "suffix" {
  byte_length = 4
}

locals {
  name_prefix = "${var.project_name}-${var.environment}"
  common_tags = merge({
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }, var.tags)
  lambda_source_path = var.lambda_source_path != "" ? var.lambda_source_path : "${path.module}/../../cmd/lambda"
}

# KMS Key for encryption
resource "aws_kms_key" "tokenization" {
  description              = "KMS key for ${local.name_prefix} tokenization"
  deletion_window_in_days  = var.kms_deletion_window
  enable_key_rotation      = var.kms_enable_rotation
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow Lambda Function"
        Effect = "Allow"
        Principal = {
          AWS = aws_iam_role.lambda_role.arn
        }
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:ReEncrypt*",
          "kms:GenerateDataKey*",
          "kms:DescribeKey"
        ]
        Resource = "*"
      }
    ]
  })

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-kms-key"
  })
}

resource "aws_kms_alias" "tokenization" {
  name          = "alias/${local.name_prefix}-${random_id.suffix.hex}"
  target_key_id = aws_kms_key.tokenization.key_id
}

# DynamoDB table for token storage
resource "aws_dynamodb_table" "tokens" {
  name           = "${local.name_prefix}-tokens-${random_id.suffix.hex}"
  billing_mode   = var.dynamodb_billing_mode
  hash_key       = "token_value"

  attribute {
    name = "token_value"
    type = "S"
  }


  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.tokenization.arn
  }

  point_in_time_recovery {
    enabled = true
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-tokens-table"
  })
}

# IAM role for Lambda function
resource "aws_iam_role" "lambda_role" {
  name = "${local.name_prefix}-lambda-role-${random_id.suffix.hex}"

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

  tags = local.common_tags
}

# IAM policy for Lambda function
resource "aws_iam_role_policy" "lambda_policy" {
  name = "${local.name_prefix}-lambda-policy-${random_id.suffix.hex}"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:log-group:/aws/lambda/${local.name_prefix}-*"
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:BatchWriteItem",
          "dynamodb:BatchGetItem",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem"
        ]
        Resource = aws_dynamodb_table.tokens.arn
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:ReEncrypt*",
          "kms:GenerateDataKey*",
          "kms:DescribeKey"
        ]
        Resource = aws_kms_key.tokenization.arn
      }
    ]
  })
}

# Attach X-Ray policy if tracing is enabled
resource "aws_iam_role_policy_attachment" "lambda_xray_policy" {
  count      = var.enable_xray_tracing ? 1 : 0
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess"
}

# CloudWatch log group for Lambda
resource "aws_cloudwatch_log_group" "lambda_logs" {
  name              = "/aws/lambda/${local.name_prefix}-lambda-${random_id.suffix.hex}"
  retention_in_days = var.log_retention_days

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-lambda-logs"
  })
}

# Lambda function
resource "aws_lambda_function" "tokenizer" {
  function_name = "${local.name_prefix}-lambda-${random_id.suffix.hex}"
  role          = aws_iam_role.lambda_role.arn
  handler       = "main"
  runtime       = var.lambda_runtime
  timeout       = var.lambda_timeout
  memory_size   = var.lambda_memory
  architectures = ["arm64"]

  filename         = "${path.module}/lambda.zip"
  source_code_hash = null_resource.package_lambda.id

  environment {
    variables = {
      DYNAMODB_TABLE_NAME = aws_dynamodb_table.tokens.name
      KMS_KEY_ID          = aws_kms_key.tokenization.key_id
      AWS_REGION          = var.aws_region
      LOG_LEVEL           = "INFO"
    }
  }

  dynamic "tracing_config" {
    for_each = var.enable_xray_tracing ? [1] : []
    content {
      mode = "Active"
    }
  }

  depends_on = [
    aws_iam_role_policy.lambda_policy,
    aws_cloudwatch_log_group.lambda_logs,
    null_resource.package_lambda,
  ]

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-lambda"
  })
}


# Data sources
data "aws_caller_identity" "current" {}

# Build the Go binary
resource "null_resource" "build_lambda" {
  triggers = {
    # Rebuild when source code changes
    source_hash = data.archive_file.source_code.output_base64sha256
  }

  provisioner "local-exec" {
    command = <<-EOT
      cd ${local.lambda_source_path}
      GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ${path.module}/main .
    EOT
  }
}

# Create source code archive for tracking changes
data "archive_file" "source_code" {
  type        = "zip"
  output_path = "${path.module}/source.zip"
  source_dir  = local.lambda_source_path
  excludes    = ["*.zip"]
}

# Create the actual deployment package
resource "null_resource" "package_lambda" {
  triggers = {
    build_hash = null_resource.build_lambda.id
  }

  provisioner "local-exec" {
    command = <<-EOT
      cd ${path.module}
      # Create bootstrap script for provided.al2023
      cat > bootstrap << 'EOF'
#!/bin/bash
set -euo pipefail
exec ./main "$@"
EOF
      chmod +x bootstrap
      # Package the Lambda function
      zip lambda.zip main bootstrap
    EOT
  }

  depends_on = [null_resource.build_lambda]
}

