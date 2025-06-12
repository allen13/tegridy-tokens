output "dynamodb_table_name" {
  description = "Name of the DynamoDB table"
  value       = module.tegridy_tokens.dynamodb_table_name
}

output "dynamodb_table_arn" {
  description = "ARN of the DynamoDB table"
  value       = module.tegridy_tokens.dynamodb_table_arn
}

output "kms_key_id" {
  description = "ID of the KMS key"
  value       = module.tegridy_tokens.kms_key_id
}

output "kms_key_arn" {
  description = "ARN of the KMS key"
  value       = module.tegridy_tokens.kms_key_arn
}

output "kms_key_alias" {
  description = "Alias of the KMS key"
  value       = module.tegridy_tokens.kms_alias_name
}

output "lambda_role_arn" {
  description = "ARN of the Lambda IAM role"
  value       = module.tegridy_tokens.lambda_role_arn
}

output "lambda_function_name" {
  description = "Name of the Lambda function"
  value       = module.tegridy_tokens.lambda_function_name
}

output "lambda_function_arn" {
  description = "ARN of the Lambda function"
  value       = module.tegridy_tokens.lambda_function_arn
}

output "aws_region" {
  description = "AWS region"
  value       = var.aws_region
}