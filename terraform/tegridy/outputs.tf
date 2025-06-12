# Lambda Function outputs
output "lambda_function_name" {
  description = "Name of the Lambda function"
  value       = aws_lambda_function.tokenizer.function_name
}

output "lambda_function_arn" {
  description = "ARN of the Lambda function"
  value       = aws_lambda_function.tokenizer.arn
}

output "lambda_function_invoke_arn" {
  description = "Invoke ARN of the Lambda function"
  value       = aws_lambda_function.tokenizer.invoke_arn
}

# DynamoDB outputs
output "dynamodb_table_name" {
  description = "Name of the DynamoDB table"
  value       = aws_dynamodb_table.tokens.name
}

output "dynamodb_table_arn" {
  description = "ARN of the DynamoDB table"
  value       = aws_dynamodb_table.tokens.arn
}

# KMS outputs
output "kms_key_id" {
  description = "ID of the KMS key"
  value       = aws_kms_key.tokenization.key_id
}

output "kms_key_arn" {
  description = "ARN of the KMS key"
  value       = aws_kms_key.tokenization.arn
}

output "kms_alias_name" {
  description = "Name of the KMS alias"
  value       = aws_kms_alias.tokenization.name
}

# IAM outputs
output "lambda_role_arn" {
  description = "ARN of the Lambda execution role"
  value       = aws_iam_role.lambda_role.arn
}

# Environment variables for application configuration
output "environment_variables" {
  description = "Environment variables needed to invoke the Lambda"
  value = {
    DYNAMODB_TABLE_NAME = aws_dynamodb_table.tokens.name
    KMS_KEY_ID          = aws_kms_key.tokenization.key_id
    AWS_REGION          = var.aws_region
  }
}