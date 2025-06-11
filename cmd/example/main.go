package main

import (
	"context"
	"fmt"
	"log"
	"time"

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

	// Create repository
	repo := repository.NewDynamoDBRepository(repository.DynamoDBConfig{
		Client:    dynamoClient,
		TableName: "tegridy-tokens",
		TTLField:  "ttl",
	})

	// Create encryptor with envelope encryption for better performance
	enc := encryption.NewKMSEncryptor(encryption.KMSConfig{
		Client:          kmsClient,
		KeyID:           "alias/tegridy-tokens", // Replace with your KMS key
		UseEnvelope:     true,
		CacheDataKeys:   true,
		CacheSize:       100,
		CacheTTLSeconds: 300, // 5 minutes
	})

	// Create tokenizer
	tkn := tokenizer.NewTokenizer(tokenizer.Config{
		Repository:     repo,
		Encryptor:      enc,
		TokenGenerator: &tokenizer.SecureTokenGenerator{},
		Workers:        20, // For batch processing
	})

	// Example 1: Single tokenization
	fmt.Println("=== Single Tokenization Example ===")
	singleExample(ctx, tkn)

	// Example 2: Batch tokenization for big data
	fmt.Println("\n=== Batch Tokenization Example ===")
	batchExample(ctx, tkn)

	// Example 3: Performance test
	fmt.Println("\n=== Performance Test ===")
	performanceTest(ctx, tkn)
}

func singleExample(ctx context.Context, tkn tokenizer.Tokenizer) {
	// Tokenize sensitive data
	req := tokenizer.TokenRequest{
		Data:       "4111-1111-1111-1111", // Example credit card
		Format:     "credit_card",
		TTLSeconds: ptr(3600), // 1 hour TTL
	}

	resp, err := tkn.Tokenize(ctx, req)
	if err != nil {
		log.Printf("tokenization error: %v", err)
		return
	}

	if !resp.Success {
		log.Printf("tokenization failed: %s", resp.Error)
		return
	}

	fmt.Printf("Original: %s\n", req.Data)
	fmt.Printf("Token: %s\n", resp.Token)

	// Detokenize
	data, err := tkn.Detokenize(ctx, resp.Token)
	if err != nil {
		log.Printf("detokenization error: %v", err)
		return
	}

	fmt.Printf("Detokenized: %s\n", data)
}

func batchExample(ctx context.Context, tkn tokenizer.Tokenizer) {
	// Create batch request with multiple items
	requests := make([]tokenizer.TokenRequest, 100)
	for i := 0; i < 100; i++ {
		requests[i] = tokenizer.TokenRequest{
			Data:       fmt.Sprintf("user-%d@example.com", i),
			Format:     "email",
			TTLSeconds: ptr(3600),
		}
	}

	batchReq := tokenizer.BatchTokenRequest{
		Requests: requests,
	}

	// Tokenize batch
	start := time.Now()
	batchResp, err := tkn.TokenizeBatch(ctx, batchReq)
	if err != nil {
		log.Printf("batch tokenization error: %v", err)
		return
	}
	elapsed := time.Since(start)

	fmt.Printf("Tokenized %d items in %v\n", len(requests), elapsed)
	fmt.Printf("Success: %d, Failed: %d\n", batchResp.Success, batchResp.Failed)
	fmt.Printf("Rate: %.2f items/second\n", float64(batchResp.Success)/elapsed.Seconds())

	// Collect tokens for detokenization
	tokens := make([]string, 0, batchResp.Success)
	for _, resp := range batchResp.Responses {
		if resp.Success {
			tokens = append(tokens, resp.Token)
		}
	}

	// Detokenize batch
	start = time.Now()
	results, err := tkn.DetokenizeBatch(ctx, tokens)
	if err != nil {
		log.Printf("batch detokenization error: %v", err)
		return
	}
	elapsed = time.Since(start)

	fmt.Printf("\nDetokenized %d items in %v\n", len(results), elapsed)
	fmt.Printf("Rate: %.2f items/second\n", float64(len(results))/elapsed.Seconds())
}

func performanceTest(ctx context.Context, tkn tokenizer.Tokenizer) {
	// Test with larger batch for big data scenarios
	batchSizes := []int{100, 500, 1000, 5000}

	for _, size := range batchSizes {
		fmt.Printf("\nTesting batch size: %d\n", size)

		// Create requests
		requests := make([]tokenizer.TokenRequest, size)
		for i := 0; i < size; i++ {
			requests[i] = tokenizer.TokenRequest{
				Data:   fmt.Sprintf("sensitive-data-%d-with-some-additional-content", i),
				Format: "custom",
			}
		}

		batchReq := tokenizer.BatchTokenRequest{
			Requests: requests,
		}

		// Measure tokenization
		start := time.Now()
		batchResp, err := tkn.TokenizeBatch(ctx, batchReq)
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}
		elapsed := time.Since(start)

		fmt.Printf("  Tokenization: %v (%.2f items/sec)\n", elapsed, float64(size)/elapsed.Seconds())

		// Collect tokens for detokenization
		tokens := make([]string, 0, batchResp.Success)
		for _, resp := range batchResp.Responses {
			if resp.Success {
				tokens = append(tokens, resp.Token)
			}
		}

		// Measure detokenization
		start = time.Now()
		_, err = tkn.DetokenizeBatch(ctx, tokens)
		if err != nil {
			log.Printf("detokenization error: %v", err)
			continue
		}
		elapsed = time.Since(start)

		fmt.Printf("  Detokenization: %v (%.2f items/sec)\n", elapsed, float64(len(tokens))/elapsed.Seconds())
	}
}

func ptr(i int64) *int64 {
	return &i
}