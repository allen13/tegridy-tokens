# Tegridy Tokens Lambda Terraform Module

This Terraform module deploys the Tegridy Tokens service as an AWS Lambda function for direct invocation (UDF pattern).

## Features

- AWS Lambda function for tokenization/detokenization
- DynamoDB table for token storage with encryption
- KMS key for envelope encryption
- IAM roles and policies with least privilege
- CloudWatch logging
- Optional X-Ray tracing

## Usage

```hcl
module "tegridy_tokens" {
  source = "./terraform/lambda"
  
  project_name = "my-project"
  environment  = "prod"
  aws_region   = "us-west-2"
  
  # Lambda configuration
  lambda_timeout = 60
  lambda_memory  = 1024
  
  # Optional: Enable X-Ray tracing
  enable_xray_tracing = true
  
  tags = {
    Department = "security"
    Owner      = "data-team"
  }
}
```

## Invoking the Lambda

The Lambda function exclusively uses batch format for all operations. This provides consistent performance, simplifies the API, and optimizes resource utilization.

### Request Format

All requests follow the same pattern:

```json
{
  "operation": "tokenize|detokenize|health",
  "data": { ... }
}
```

### Supported Operations

- `tokenize`: Tokenize sensitive data (batch format only)
- `detokenize`: Detokenize tokens back to original data (batch format only)
- `health`: Health check

### Tokenization

Tokenization requests must use the batch format with a `requests` array, even for single items:

```json
{
  "operation": "tokenize",
  "data": {
    "requests": [
      {
        "data": "4111-1111-1111-1111",
        "preserve_format": true
      },
      {
        "data": "123-45-6789",
        "preserve_format": true,
        "format_hint": "ssn"
      }
    ]
  }
}
```

### Detokenization

Detokenization requests must use the batch format with a `tokens` array, even for single tokens:

```json
{
  "operation": "detokenize",
  "data": {
    "tokens": ["7561-6533-6533-6533", "123-45-6789"]
  }
}
```

### Example Invocations (AWS CLI)

```bash
# Tokenize a single item (must use batch format)
aws lambda invoke \
  --function-name "$(terraform output -raw lambda_function_name)" \
  --payload '{
    "operation": "tokenize",
    "data": {
      "requests": [{
        "data": "4111-1111-1111-1111",
        "preserve_format": true
      }]
    }
  }' \
  response.json

# Tokenize multiple items
aws lambda invoke \
  --function-name "$(terraform output -raw lambda_function_name)" \
  --payload '{
    "operation": "tokenize",
    "data": {
      "requests": [
        {"data": "4111-1111-1111-1111", "preserve_format": true},
        {"data": "123-45-6789", "preserve_format": true, "format_hint": "ssn"}
      ]
    }
  }' \
  response.json

# Detokenize a single token (must use batch format)
aws lambda invoke \
  --function-name "$(terraform output -raw lambda_function_name)" \
  --payload '{
    "operation": "detokenize",
    "data": {
      "tokens": ["7561-6533-6533-6533"]
    }
  }' \
  response.json

# Detokenize multiple tokens
aws lambda invoke \
  --function-name "$(terraform output -raw lambda_function_name)" \
  --payload '{
    "operation": "detokenize",
    "data": {
      "tokens": ["7561-6533-6533-6533", "123-45-6789"]
    }
  }' \
  response.json

# Health check
aws lambda invoke \
  --function-name "$(terraform output -raw lambda_function_name)" \
  --payload '{"operation": "health"}' \
  response.json
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| aws | ~> 5.0 |
| archive | ~> 2.4 |
| random | ~> 3.6 |
| null | ~> 3.2 |

## Providers

| Name | Version |
|------|---------|
| aws | ~> 5.0 |
| archive | ~> 2.4 |
| random | ~> 3.6 |
| null | ~> 3.2 |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| project_name | Name of the project | `string` | `"tegridy-tokens"` | no |
| environment | Environment (dev, staging, prod) | `string` | `"dev"` | no |
| aws_region | AWS region | `string` | `"us-east-1"` | no |
| lambda_timeout | Lambda function timeout in seconds | `number` | `30` | no |
| lambda_memory | Lambda function memory in MB | `number` | `512` | no |
| lambda_runtime | Lambda runtime version | `string` | `"provided.al2"` | no |
| dynamodb_billing_mode | DynamoDB billing mode | `string` | `"PAY_PER_REQUEST"` | no |
| dynamodb_ttl_enabled | Enable TTL for DynamoDB table | `bool` | `true` | no |
| kms_deletion_window | KMS key deletion window in days | `number` | `7` | no |
| kms_enable_rotation | Enable automatic KMS key rotation | `bool` | `true` | no |
| enable_xray_tracing | Enable X-Ray tracing for Lambda | `bool` | `true` | no |
| log_retention_days | CloudWatch log retention in days | `number` | `14` | no |
| tags | Additional tags for resources | `map(string)` | `{}` | no |

## Outputs

| Name | Description |
|------|-------------|
| lambda_function_name | Name of the Lambda function |
| lambda_function_arn | ARN of the Lambda function |
| lambda_function_invoke_arn | Invoke ARN of the Lambda function |
| dynamodb_table_name | Name of the DynamoDB table |
| dynamodb_table_arn | ARN of the DynamoDB table |
| kms_key_id | ID of the KMS key |
| kms_key_arn | ARN of the KMS key |
| kms_alias_name | Name of the KMS alias |
| lambda_role_arn | ARN of the Lambda execution role |
| environment_variables | Environment variables needed to invoke the Lambda |

## Architecture

The module creates the following AWS resources:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Lambda        │    │   DynamoDB      │    │      KMS        │
│   Function      │───▶│     Table       │    │     Key         │
│                 │    │                 │◀───│                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CloudWatch    │    │      IAM        │    │     Random      │
│     Logs        │    │     Roles       │    │    Suffix       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Security

- All data is encrypted at rest using KMS
- IAM roles follow least privilege principle
- Lambda function runs with minimal required permissions
- DynamoDB table is encrypted with customer-managed KMS key
- CloudWatch logs are retained for configurable period

## Building and Deployment

The module automatically:

1. Builds the Go binary for Linux/AMD64
2. Creates a deployment package with the binary and bootstrap script
3. Deploys the Lambda function with the package

Make sure you have Go installed and can cross-compile for Linux.

## Performance Considerations

- Default memory allocation is 512MB (adjustable)
- Connection pooling is handled by AWS SDK
- Envelope encryption reduces KMS API calls
- Batch operations optimize DynamoDB usage

## Monitoring

- CloudWatch logs are automatically configured
- Optional X-Ray tracing for performance insights
- Lambda metrics available in CloudWatch

## Cost Optimization

- Pay-per-request billing for DynamoDB
- Lambda charges only for execution time
- KMS charges per API call (optimized with envelope encryption)
- CloudWatch logs retention configurable