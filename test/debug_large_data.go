package main

import (
	"context"
	"fmt"
	"log"

	"github.com/allen13/tegridy-tokens/pkg/encryption"
	"github.com/allen13/tegridy-tokens/pkg/repository"
	"github.com/allen13/tegridy-tokens/pkg/tokenizer"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

func main() {
	ctx := context.Background()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	// Create AWS clients
	dynamoClient := dynamodb.NewFromConfig(cfg)
	kmsClient := kms.NewFromConfig(cfg)

	// Create repository (using a dummy table name - this won't work without real infrastructure)
	repo := repository.NewDynamoDBRepository(repository.DynamoDBConfig{
		Client:    dynamoClient,
		TableName: "dummy-table",
		TTLField:  "ttl",
	})

	// Create encryptor with envelope encryption
	enc := encryption.NewKMSEncryptor(encryption.KMSConfig{
		Client:      kmsClient,
		KeyID:       "dummy-key",
		UseEnvelope: true,
	})

	// Create tokenizer
	tkn := tokenizer.NewTokenizer(tokenizer.Config{
		Repository: repo,
		Encryptor:  enc,
	})

	// Test large data generation
	size := 1024 * 1024 // 1 MB
	data := generateRandomString(size)
	
	fmt.Printf("Generated data size: %d bytes\n", len(data))
	fmt.Printf("First 100 characters: %s\n", data[:100])
	
	// Try to tokenize (this will fail due to dummy credentials, but we can check the data)
	req := tokenizer.TokenRequest{
		Data: data,
	}
	
	resp, err := tkn.Tokenize(ctx, req)
	if err != nil {
		fmt.Printf("Expected error due to dummy credentials: %v\n", err)
	} else {
		fmt.Printf("Response: %+v\n", resp)
	}
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}