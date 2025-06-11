package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/allen13/tegridy-tokens/pkg/tokenizer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBRepository implements the Repository interface using DynamoDB
type DynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
	ttlField  string
}

// DynamoDBConfig holds configuration for DynamoDB repository
type DynamoDBConfig struct {
	Client    *dynamodb.Client
	TableName string
	TTLField  string // Optional: field name for TTL, default "ttl"
}

// TokenRecord represents a token record in DynamoDB
type TokenRecord struct {
	TokenValue    string `dynamodbav:"token_value"`
	TokenID       string `dynamodbav:"token_id"`
	EncryptedData []byte `dynamodbav:"encrypted_data"`
	CreatedAt     int64  `dynamodbav:"created_at"`
	TTL           *int64 `dynamodbav:"ttl,omitempty"`
}

// NewDynamoDBRepository creates a new DynamoDB repository
func NewDynamoDBRepository(cfg DynamoDBConfig) tokenizer.Repository {
	ttlField := cfg.TTLField
	if ttlField == "" {
		ttlField = "ttl"
	}
	
	return &DynamoDBRepository{
		client:    cfg.Client,
		tableName: cfg.TableName,
		ttlField:  ttlField,
	}
}

// Store saves an encrypted token
func (r *DynamoDBRepository) Store(ctx context.Context, token *tokenizer.Token, encryptedData []byte) error {
	record := TokenRecord{
		TokenValue:    token.Value,
		TokenID:       token.ID,
		EncryptedData: encryptedData,
		CreatedAt:     token.CreatedAt.Unix(),
	}
	
	// Calculate TTL if provided
	if token.TTL != nil && *token.TTL > 0 {
		ttl := time.Now().Unix() + *token.TTL
		record.TTL = &ttl
	}
	
	av, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal token record: %w", err)
	}
	
	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	}
	
	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}
	
	return nil
}

// Retrieve gets encrypted data by token
func (r *DynamoDBRepository) Retrieve(ctx context.Context, tokenValue string) ([]byte, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"token_value": &types.AttributeValueMemberS{Value: tokenValue},
		},
	}
	
	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}
	
	if result.Item == nil {
		return nil, fmt.Errorf("token not found")
	}
	
	var record TokenRecord
	err = attributevalue.UnmarshalMap(result.Item, &record)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token record: %w", err)
	}
	
	return record.EncryptedData, nil
}

// StoreBatch saves multiple tokens efficiently
func (r *DynamoDBRepository) StoreBatch(ctx context.Context, items []tokenizer.TokenItem) error {
	// DynamoDB BatchWriteItem has a limit of 25 items per request
	const batchSize = 25
	
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		
		if err := r.storeBatchChunk(ctx, items[i:end]); err != nil {
			return err
		}
	}
	
	return nil
}

// storeBatchChunk stores a chunk of items (max 25)
func (r *DynamoDBRepository) storeBatchChunk(ctx context.Context, items []tokenizer.TokenItem) error {
	writeRequests := make([]types.WriteRequest, 0, len(items))
	
	for _, item := range items {
		record := TokenRecord{
			TokenValue:    item.Token.Value,
			TokenID:       item.Token.ID,
			EncryptedData: item.EncryptedData,
			CreatedAt:     item.Token.CreatedAt.Unix(),
		}
		
		// Calculate TTL if provided
		if item.Token.TTL != nil && *item.Token.TTL > 0 {
			ttl := time.Now().Unix() + *item.Token.TTL
			record.TTL = &ttl
		}
		
		av, err := attributevalue.MarshalMap(record)
		if err != nil {
			return fmt.Errorf("failed to marshal token record: %w", err)
		}
		
		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: av,
			},
		})
	}
	
	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			r.tableName: writeRequests,
		},
	}
	
	// Execute batch write with retry for unprocessed items
	for {
		result, err := r.client.BatchWriteItem(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to batch write items: %w", err)
		}
		
		// Check for unprocessed items
		if len(result.UnprocessedItems) == 0 {
			break
		}
		
		// Retry unprocessed items
		input.RequestItems = result.UnprocessedItems
		
		// Add a small delay before retry
		time.Sleep(100 * time.Millisecond)
	}
	
	return nil
}

// RetrieveBatch gets multiple tokens efficiently
func (r *DynamoDBRepository) RetrieveBatch(ctx context.Context, tokens []string) (map[string][]byte, error) {
	results := make(map[string][]byte)
	
	// DynamoDB BatchGetItem has a limit of 100 items per request
	const batchSize = 100
	
	for i := 0; i < len(tokens); i += batchSize {
		end := i + batchSize
		if end > len(tokens) {
			end = len(tokens)
		}
		
		batchResults, err := r.retrieveBatchChunk(ctx, tokens[i:end])
		if err != nil {
			return nil, err
		}
		
		// Merge results
		for k, v := range batchResults {
			results[k] = v
		}
	}
	
	return results, nil
}

// retrieveBatchChunk retrieves a chunk of tokens (max 100)
func (r *DynamoDBRepository) retrieveBatchChunk(ctx context.Context, tokens []string) (map[string][]byte, error) {
	keys := make([]map[string]types.AttributeValue, 0, len(tokens))
	
	for _, token := range tokens {
		keys = append(keys, map[string]types.AttributeValue{
			"token_value": &types.AttributeValueMemberS{Value: token},
		})
	}
	
	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			r.tableName: {
				Keys: keys,
			},
		},
	}
	
	results := make(map[string][]byte)
	
	// Execute batch get with retry for unprocessed keys
	for {
		result, err := r.client.BatchGetItem(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to batch get items: %w", err)
		}
		
		// Process responses
		if responses, ok := result.Responses[r.tableName]; ok {
			for _, item := range responses {
				var record TokenRecord
				err = attributevalue.UnmarshalMap(item, &record)
				if err != nil {
					continue // Skip failed unmarshals
				}
				results[record.TokenValue] = record.EncryptedData
			}
		}
		
		// Check for unprocessed keys
		if len(result.UnprocessedKeys) == 0 {
			break
		}
		
		// Retry unprocessed keys
		input.RequestItems = result.UnprocessedKeys
		
		// Add a small delay before retry
		time.Sleep(100 * time.Millisecond)
	}
	
	return results, nil
}