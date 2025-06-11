output "dynamodb_table_name" {
  description = "Name of the DynamoDB table"
  value       = aws_dynamodb_table.tokens.name
}

output "dynamodb_table_arn" {
  description = "ARN of the DynamoDB table"
  value       = aws_dynamodb_table.tokens.arn
}

output "kms_key_id" {
  description = "ID of the KMS key"
  value       = aws_kms_key.tokenization.id
}

output "kms_key_arn" {
  description = "ARN of the KMS key"
  value       = aws_kms_key.tokenization.arn
}

output "kms_key_alias" {
  description = "Alias of the KMS key"
  value       = aws_kms_alias.tokenization.name
}

output "lambda_role_arn" {
  description = "ARN of the Lambda IAM role"
  value       = aws_iam_role.lambda_role.arn
}

output "aws_region" {
  description = "AWS region"
  value       = var.aws_region
}