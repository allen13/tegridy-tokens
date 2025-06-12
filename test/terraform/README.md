# Tegridy Tokens Test Infrastructure

This directory contains Terraform configuration for integration testing the Tegridy Tokens library.

## Overview

The test infrastructure uses the production Lambda module from `../../terraform/lambda` to ensure tests run against the same infrastructure configuration that would be deployed in production.

## Usage

The Terraform configuration is primarily used by Terratest for automated integration testing. It deploys:

- The complete Lambda module including:
  - Lambda function for tokenization/detokenization
  - DynamoDB table for token storage
  - KMS key for encryption
  - IAM roles and policies
  - CloudWatch logs

## Configuration

The test configuration uses sensible defaults optimized for testing:

- **Environment**: `test`
- **Project Name**: `tegridy-test`
- **Lambda Memory**: 512MB
- **Lambda Timeout**: 30 seconds
- **DynamoDB Billing**: Pay-per-request (no pre-provisioned capacity)
- **Log Retention**: 1 day (to minimize costs)
- **KMS Key Rotation**: Disabled (for testing)
- **X-Ray Tracing**: Disabled (for testing)

## Running Tests

Tests are typically run via Terratest from the parent directory:

```bash
cd ../
go test -v -timeout 30m
```

## Manual Deployment (for debugging)

If you need to manually deploy the test infrastructure:

```bash
terraform init
terraform plan -var="aws_region=us-east-1"
terraform apply -var="aws_region=us-east-1"
```

Remember to destroy the resources when done:

```bash
terraform destroy -var="aws_region=us-east-1"
```

## Cost Considerations

The test infrastructure is designed to minimize costs:
- Uses pay-per-request DynamoDB billing
- Minimal Lambda memory allocation
- Short log retention period
- Resources are automatically cleaned up by Terratest

## Important Notes

- All resources are tagged with `ManagedBy=terratest` for easy identification
- Resources include a random suffix to avoid naming conflicts
- The infrastructure is automatically destroyed after tests complete